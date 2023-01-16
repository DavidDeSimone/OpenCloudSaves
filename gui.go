package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/webview/webview"
)

//go:embed html/index.html
var htmlMain embed.FS

//go:embed html/style.css
var cssContent string

//go:embed html/main.js
var jsContent string

type SaveFile struct {
	Filename   string
	ModifiedBy string
	Size       string
}

type Game struct {
	Name      string
	Def       *GameDef
	SaveFiles []SaveFile
}

type HtmlInput struct {
	Games     []Game
	Platforms []string
}

func consoleLog(s string) {
	fmt.Println(s)
}

var chanelMutex sync.Mutex
var channelMap map[string]*ChannelProvider = make(map[string]*ChannelProvider)

func pollProgress(key string) (*ProgressEvent, error) {
	chanelMutex.Lock()
	channels, ok := channelMap[key]
	chanelMutex.Unlock()

	if !ok {
		return nil, fmt.Errorf("failed to find progress event")
	}

	result := &ProgressEvent{}
	select {
	case res := <-channels.progress:
		result = &res
	default:
		//no-op
	}

	return result, nil
}

func syncGame(key string) {
	ops := &Options{
		Gamenames: []string{key},
	}
	dm := MakeGameDefManager("")
	channels := &ChannelProvider{
		logs:     make(chan Message, 100),
		cancel:   make(chan Cancellation, 1),
		input:    make(chan SyncRequest, 10),
		output:   make(chan SyncResponse, 10),
		progress: make(chan ProgressEvent, 20),
	}

	// go consoleLogger(channels.logs)
	go CliMain(ops, dm, channels, SyncOp)
	chanelMutex.Lock()
	defer chanelMutex.Unlock()

	channelMap[key] = channels

	// @TODO this needs to yield to JS. Instead, we should have this be a go thread
	// and then JS needs to poll that channel on main thread.
	// for {
	// 	select {
	// 	case res := <-channels.progress:
	// 		fmt.Println(res)
	// 	case <-time.After(1 * time.Second):
	// 		fmt.Println("timeout 1")
	// 	}

	// 	// msg := <-channels.logs
	// 	// if msg.Finished {
	// 	// 	break
	// 	// } else {
	// 	// 	fmt.Println(msg.Message)
	// 	// }
	// }
}

type GuiDatapath struct {
	Path     string
	Exts     []string
	Ignore   []string
	Download bool
	Upload   bool
	Delete   bool
}

type GuiGamedef struct {
	Name    string
	Windows GuiDatapath
	MacOS   GuiDatapath
	Linux   GuiDatapath
}

func removeGamedefByKey(key string) {
	dm := MakeGameDefManager("")
	gamedefMap := dm.GetGameDefMap()
	delete(gamedefMap, key)
	dm.CommitUserOverrides()
}

func fetchGamedef(key string) (*GuiGamedef, error) {
	dm := MakeGameDefManager("")
	gamedefMap := dm.GetGameDefMap()
	def, ok := gamedefMap[key]
	if !ok {
		return nil, fmt.Errorf("gamedef not found")
	}

	result := &GuiGamedef{
		Name: def.DisplayName,
	}

	if len(def.WinPath) > 0 {
		path := def.WinPath[0]
		result.Windows = GuiDatapath{
			Path:     path.Path,
			Exts:     path.Exts,
			Ignore:   path.Ignore,
			Download: path.NetAuth&CloudOperationDownload != 0,
			Upload:   path.NetAuth&CloudOperationUpload != 0,
			Delete:   path.NetAuth&CloudOperationDelete != 0,
		}
	}

	if len(def.DarwinPath) > 0 {
		path := def.DarwinPath[0]
		result.MacOS = GuiDatapath{
			Path:     path.Path,
			Exts:     path.Exts,
			Ignore:   path.Ignore,
			Download: path.NetAuth&CloudOperationDownload != 0,
			Upload:   path.NetAuth&CloudOperationUpload != 0,
			Delete:   path.NetAuth&CloudOperationDelete != 0,
		}
	}

	if len(def.LinuxPath) > 0 {
		path := def.LinuxPath[0]
		result.Linux = GuiDatapath{
			Path:     path.Path,
			Exts:     path.Exts,
			Ignore:   path.Ignore,
			Download: path.NetAuth&CloudOperationDownload != 0,
			Upload:   path.NetAuth&CloudOperationUpload != 0,
			Delete:   path.NetAuth&CloudOperationDelete != 0,
		}
	}

	return result, nil
}

func commitGamedef(gamedef GuiGamedef) {
	dm := MakeGameDefManager("")
	gamedefMap := dm.GetGameDefMap()
	gamedefMap[gamedef.Name] = &GameDef{
		DisplayName: gamedef.Name,
		SteamId:     "0",
	}

	netauth := 0
	if gamedef.Windows.Download {
		netauth |= CloudOperationDownload
	}
	if gamedef.Windows.Upload {
		netauth |= CloudOperationUpload
	}
	if gamedef.Windows.Delete {
		netauth |= CloudOperationDelete
	}

	list := strings.Split(gamedef.Windows.Path, string(os.PathSeparator))
	parent := ""
	if len(list) == 0 {
		parent = gamedef.Windows.Path
	} else {
		parent = list[len(list)-1]
	}

	gamedefMap[gamedef.Name].WinPath = []*Datapath{
		{
			Path:    gamedef.Windows.Path,
			Exts:    gamedef.Windows.Exts,
			Ignore:  gamedef.Windows.Ignore,
			Parent:  parent,
			NetAuth: netauth,
		},
	}

	netauth = 0
	if gamedef.MacOS.Download {
		netauth |= CloudOperationDownload
	}
	if gamedef.MacOS.Upload {
		netauth |= CloudOperationUpload
	}
	if gamedef.MacOS.Delete {
		netauth |= CloudOperationDelete
	}

	list = strings.Split(gamedef.MacOS.Path, string(os.PathSeparator))
	parent = ""
	if len(list) == 0 {
		parent = gamedef.MacOS.Path
	} else {
		parent = list[len(list)-1]
	}

	gamedefMap[gamedef.Name].DarwinPath = []*Datapath{
		{
			Path:    gamedef.MacOS.Path,
			Exts:    gamedef.MacOS.Exts,
			Ignore:  gamedef.MacOS.Ignore,
			Parent:  parent,
			NetAuth: netauth,
		},
	}

	netauth = 0
	if gamedef.Linux.Download {
		netauth |= CloudOperationDownload
	}
	if gamedef.Linux.Upload {
		netauth |= CloudOperationUpload
	}
	if gamedef.Linux.Delete {
		netauth |= CloudOperationDelete
	}

	list = strings.Split(gamedef.Linux.Path, string(os.PathSeparator))
	parent = ""
	if len(list) == 0 {
		parent = gamedef.Linux.Path
	} else {
		parent = list[len(list)-1]
	}

	gamedefMap[gamedef.Name].LinuxPath = []*Datapath{
		{
			Path:    gamedef.Linux.Path,
			Exts:    gamedef.Linux.Exts,
			Ignore:  gamedef.Linux.Ignore,
			Parent:  parent,
			NetAuth: netauth,
		},
	}

	dm.CommitUserOverrides()
}

func bindFunctions(w webview.WebView) {
	w.Bind("log", consoleLog)
	w.Bind("syncGame", syncGame)
	w.Bind("refresh", func() {
		refreshMainContent(w)
	})
	w.Bind("commitGamedef", commitGamedef)
	w.Bind("removeGamedefByKey", removeGamedefByKey)
	w.Bind("fetchGamedef", fetchGamedef)
	w.Bind("pollProgress", pollProgress)
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func buildGamelist(dm GameDefManager) []Game {
	games := []Game{}
	for k, v := range dm.GetGameDefMap() {
		game := Game{
			Name: k,
			Def:  v,
		}

		game.SaveFiles = []SaveFile{}
		paths, err := v.GetSyncpaths()
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, datapath := range paths {
			fmt.Println("Datapath " + datapath.Path)
			// @TODO better handle empty path being root
			// because of the logic in GetSyncpaths
			if datapath.Path == "" || datapath.Path == "/" {
				continue
			}

			dirFiles, err := os.ReadDir(datapath.Path)
			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, dirFile := range dirFiles {
				fmt.Println("Examining " + dirFile.Name())
				if SyncFilter(dirFile.Name(), datapath) {
					continue
				}

				info, err := dirFile.Info()
				if err != nil {
					fmt.Println(err)
					continue
				}

				size := info.Size()
				if info.IsDir() {
					size, err = DirSize(datapath.Path + string(os.PathSeparator) + info.Name())
					if err != nil {
						fmt.Println(err)
						continue
					}
				}

				game.SaveFiles = append(game.SaveFiles, SaveFile{
					Filename:   info.Name(),
					ModifiedBy: info.ModTime().Format(time.RFC3339),
					Size:       fmt.Sprintf("%vMB", size/(1024*1024)),
				})
			}
		}

		games = append(games, game)
	}

	return games
}

func executeTemplate() (string, error) {
	dm := MakeGameDefManager("")
	games := buildGamelist(dm)

	sort.Slice(games, func(i, j int) bool {
		return games[i].Def.DisplayName < games[j].Def.DisplayName
	})
	input := HtmlInput{
		Games:     games,
		Platforms: []string{"Windows", "MacOS", "Linux"},
	}

	var b bytes.Buffer
	htmlWriter := bufio.NewWriterSize(&b, 2*1024*1024)

	templ := template.Must(template.ParseFS(htmlMain, "html/index.html"))
	err := templ.Execute(htmlWriter, input)
	if err != nil {
		return "", err
	}

	htmlWriter.Flush()
	result := b.String()
	js := fmt.Sprintf("<script>%v</script>", jsContent)
	css := fmt.Sprintf("<style>%v</style>", cssContent)
	finalResult := css + result + js
	// fmt.Println(result)
	return finalResult, nil
}

func refreshMainContent(w webview.WebView) error {
	html, err := executeTemplate()
	if err != nil {
		return err
	}

	w.SetHtml(html)
	return nil
}

func GuiMain(ops *Options, dm GameDefManager) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Steam Custom Cloud Uploads")
	w.SetSize(800, 600, webview.HintNone)
	bindFunctions(w)
	err := refreshMainContent(w)
	if err != nil {
		log.Fatal(err)
	}
	w.Run()
}
