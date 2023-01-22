package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sqweek/dialog"
	"github.com/webview/webview"
)

//go:embed html
var html embed.FS

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
	Name           string
	Def            *GameDef
	SaveFiles      []SaveFile
	SaveFilesFound bool
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

func openDirDialog() (string, error) {
	return dialog.Directory().Title("Select Folder").Browse()
}

func pollLogs(key string) (string, error) {
	chanelMutex.Lock()
	channels, ok := channelMap[key]
	chanelMutex.Unlock()
	if !ok {
		return "", fmt.Errorf("failed to find progress event")
	}

	select {
	case res := <-channels.logs:
		if res.Finished {
			return "finished", nil
		} else if res.Err != nil {
			return "", res.Err
		} else {
			return res.Message, nil
		}
	default:
		//no-op
	}

	return "", nil
}

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

	srv := GetDefaultService()

	go CliMain(srv, ops, dm, channels, SyncOp)
	chanelMutex.Lock()
	defer chanelMutex.Unlock()

	channelMap[key] = channels
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
	Windows []GuiDatapath
	MacOS   []GuiDatapath
	Linux   []GuiDatapath
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

	resultDef := &GuiGamedef{
		Name: def.DisplayName,
	}

	for _, path := range def.WinPath {
		resultDef.Windows = append(resultDef.Windows, GuiDatapath{
			Path:     path.Path,
			Exts:     path.Exts,
			Ignore:   path.Ignore,
			Download: path.NetAuth&CloudOperationDownload != 0,
			Upload:   path.NetAuth&CloudOperationUpload != 0,
			Delete:   path.NetAuth&CloudOperationDelete != 0,
		})
	}

	for _, path := range def.DarwinPath {
		resultDef.MacOS = append(resultDef.MacOS, GuiDatapath{
			Path:     path.Path,
			Exts:     path.Exts,
			Ignore:   path.Ignore,
			Download: path.NetAuth&CloudOperationDownload != 0,
			Upload:   path.NetAuth&CloudOperationUpload != 0,
			Delete:   path.NetAuth&CloudOperationDelete != 0,
		})
	}

	for _, path := range def.LinuxPath {
		resultDef.Linux = append(resultDef.Linux, GuiDatapath{
			Path:     path.Path,
			Exts:     path.Exts,
			Ignore:   path.Ignore,
			Download: path.NetAuth&CloudOperationDownload != 0,
			Upload:   path.NetAuth&CloudOperationUpload != 0,
			Delete:   path.NetAuth&CloudOperationDelete != 0,
		})
	}

	return resultDef, nil
}

func commitGamedef(gamedef GuiGamedef) {
	dm := MakeGameDefManager("")
	gamedefMap := dm.GetGameDefMap()
	gamedefMap[gamedef.Name] = &GameDef{
		DisplayName: gamedef.Name,
		SteamId:     "0",
		WinPath:     []*Datapath{},
		DarwinPath:  []*Datapath{},
		LinuxPath:   []*Datapath{},
	}

	for _, def := range gamedef.Windows {
		netauth := 0
		if def.Download {
			netauth |= CloudOperationDownload
		}
		if def.Upload {
			netauth |= CloudOperationUpload
		}
		if def.Delete {
			netauth |= CloudOperationDelete
		}

		list := strings.Split(def.Path, string(os.PathSeparator))
		parent := ""
		if len(list) == 0 {
			parent = def.Path
		} else {
			parent = list[len(list)-1]
		}

		gamedefMap[gamedef.Name].WinPath = append(gamedefMap[gamedef.Name].WinPath, &Datapath{
			Path:    def.Path,
			Exts:    def.Exts,
			Ignore:  def.Ignore,
			Parent:  parent,
			NetAuth: netauth,
		})
	}

	for _, def := range gamedef.MacOS {
		netauth := 0
		if def.Download {
			netauth |= CloudOperationDownload
		}
		if def.Upload {
			netauth |= CloudOperationUpload
		}
		if def.Delete {
			netauth |= CloudOperationDelete
		}

		list := strings.Split(def.Path, string(os.PathSeparator))
		parent := ""
		if len(list) == 0 {
			parent = def.Path
		} else {
			parent = list[len(list)-1]
		}

		gamedefMap[gamedef.Name].DarwinPath = append(gamedefMap[gamedef.Name].DarwinPath, &Datapath{
			Path:    def.Path,
			Exts:    def.Exts,
			Ignore:  def.Ignore,
			Parent:  parent,
			NetAuth: netauth,
		})
	}

	for _, def := range gamedef.Linux {
		netauth := 0
		if def.Download {
			netauth |= CloudOperationDownload
		}
		if def.Upload {
			netauth |= CloudOperationUpload
		}
		if def.Delete {
			netauth |= CloudOperationDelete
		}

		list := strings.Split(def.Path, string(os.PathSeparator))
		parent := ""
		if len(list) == 0 {
			parent = def.Path
		} else {
			parent = list[len(list)-1]
		}

		gamedefMap[gamedef.Name].LinuxPath = append(gamedefMap[gamedef.Name].LinuxPath, &Datapath{
			Path:    def.Path,
			Exts:    def.Exts,
			Ignore:  def.Ignore,
			Parent:  parent,
			NetAuth: netauth,
		})
	}

	dm.CommitUserOverrides()
}

func load(w webview.WebView, path string) error {
	b, err := fs.ReadFile(html, path)
	if err != nil {
		return err
	}

	w.Eval(string(b))
	return nil
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
	w.Bind("pollLogs", pollLogs)
	w.Bind("require", func(path string) {
		load(w, path)
	})
	w.Bind("openDirDialog", openDirDialog)
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

		if len(game.SaveFiles) > 0 {
			game.SaveFilesFound = true
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
	w.SetTitle("Open Cloud Save")
	w.SetSize(800, 600, 0)
	bindFunctions(w)
	err := refreshMainContent(w)
	if err != nil {
		log.Fatal(err)
	}
	w.Run()
}
