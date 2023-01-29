package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/sqweek/dialog"
	"github.com/webview/webview"
)

// @TODO merge everything to the embedded html synthetic fs

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
		if res.Err != nil {
			return "", res.Err
		}

		resultJson, err := json.Marshal(res)
		if err != nil {
			return "", err
		}

		return string(resultJson), nil
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
		Verbose:   []bool{true},
	}
	cm := MakeCloudManager()

	dm := MakeGameDefManager("")
	dm.SetCloudManager(cm)
	channels := &ChannelProvider{
		logs:     make(chan Message, 100),
		cancel:   make(chan Cancellation, 1),
		progress: make(chan ProgressEvent, 20),
	}

	go CliMain(cm, ops, dm, channels)
	chanelMutex.Lock()
	defer chanelMutex.Unlock()

	channelMap[key] = channels
}

type GuiDatapath struct {
	Path    string
	Include string
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
			Path:    path.Path,
			Include: path.Include,
		})
	}

	for _, path := range def.DarwinPath {
		resultDef.MacOS = append(resultDef.MacOS, GuiDatapath{
			Path:    path.Path,
			Include: path.Include,
		})
	}

	for _, path := range def.LinuxPath {
		resultDef.Linux = append(resultDef.Linux, GuiDatapath{
			Path:    path.Path,
			Include: path.Include,
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
		list := strings.Split(def.Path, string(os.PathSeparator))
		parent := ""
		if len(list) == 0 {
			parent = def.Path
		} else {
			parent = list[len(list)-1]
		}

		gamedefMap[gamedef.Name].WinPath = append(gamedefMap[gamedef.Name].WinPath, &Datapath{
			Path:    def.Path,
			Include: def.Include,
			Parent:  parent,
		})
	}

	for _, def := range gamedef.MacOS {
		list := strings.Split(def.Path, string(os.PathSeparator))
		parent := ""
		if len(list) == 0 {
			parent = def.Path
		} else {
			parent = list[len(list)-1]
		}

		gamedefMap[gamedef.Name].DarwinPath = append(gamedefMap[gamedef.Name].DarwinPath, &Datapath{
			Path:    def.Path,
			Include: def.Include,
			Parent:  parent,
		})
	}

	for _, def := range gamedef.Linux {
		list := strings.Split(def.Path, string(os.PathSeparator))
		parent := ""
		if len(list) == 0 {
			parent = def.Path
		} else {
			parent = list[len(list)-1]
		}

		gamedefMap[gamedef.Name].LinuxPath = append(gamedefMap[gamedef.Name].LinuxPath, &Datapath{
			Path:    def.Path,
			Include: def.Include,
			Parent:  parent,
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

func commitCloudService(service int) error {
	cloudperfs := GetCurrentCloudPerfsOrDefault()
	cloudperfs.Cloud = service
	return CommitCloudPerfs(cloudperfs)
}

func getCloudService() (int, error) {
	cloudperfs, err := GetCurrentCloudPerfs()
	if err != nil {
		return -1, nil
	}

	return cloudperfs.Cloud, nil
}

func getSyncDryRun(name string) error {
	ops := &Options{
		DryRun:    []bool{true},
		Gamenames: []string{name},
		Verbose:   []bool{true},
	}
	channels := &ChannelProvider{
		logs:     make(chan Message, 100),
		progress: make(chan ProgressEvent, 15),
	}

	chanelMutex.Lock()
	channelMap[name] = channels
	chanelMutex.Unlock()

	cm := MakeCloudManager()
	dm := MakeGameDefManager("")
	dm.SetCloudManager(cm)

	go CliMain(cm, ops, dm, channels)
	return nil
}

func getShouldPerformDryRun() (bool, error) {
	cloudperfs := GetCurrentCloudPerfsOrDefault()
	return cloudperfs.PerformDryRun, nil
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
	w.Bind("commitCloudService", commitCloudService)
	w.Bind("setCloudSelectScreen", func() error {
		return setCloudSelectScreen(w)
	})
	w.Bind("getCloudService", getCloudService)
	w.Bind("getSyncDryRun", getSyncDryRun)
	w.Bind("getShouldPerformDryRun", getShouldPerformDryRun)
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
	cm := MakeCloudManager()
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

			cloudops := GetDefaultCloudOptions()
			cloudops.Include = datapath.Include
			files, err := cm.ListFiles(cloudops, datapath.Path)
			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, file := range files {
				if file.IsDir {
					continue
				}

				game.SaveFiles = append(game.SaveFiles,
					SaveFile{
						Filename:   file.Name,
						ModifiedBy: file.ModTime,
						Size:       fmt.Sprintf("%vMB", file.Size/(1024*1024)),
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

	syncgamejsbytes, err := fs.ReadFile(html, "html/syncgame.js")
	if err != nil {
		return "", err
	}

	htmlWriter.Flush()
	result := b.String()
	js := fmt.Sprintf("<script>%v</script>", jsContent)
	css := fmt.Sprintf("<style>%v</style>", cssContent)
	syncgamejs := fmt.Sprintf("\n<script>%v</script>\n", string(syncgamejsbytes))
	finalResult := css + result + js + syncgamejs
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

func setCloudSelectScreen(w webview.WebView) error {
	htmlbytes, err := fs.ReadFile(html, "html/selectcloud.html")
	if err != nil {
		return err
	}

	jsbytes, err := fs.ReadFile(html, "html/selectcloud.js")
	if err != nil {
		return err
	}

	cssbytes, err := fs.ReadFile(html, "html/selectcloud.css")
	if err != nil {
		return err
	}

	htmlcontent := fmt.Sprintf("<style>%v</style>", string(cssbytes)) + string(htmlbytes) + fmt.Sprintf("<script>%v</script>", string(jsbytes))
	w.SetHtml(htmlcontent)
	return nil
}

func GuiMain(ops *Options, dm GameDefManager) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Open Cloud Save")
	w.SetSize(800, 600, 0)
	bindFunctions(w)

	storage := GetCurrentStorageProvider()
	if storage == nil {
		err := setCloudSelectScreen(w)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := refreshMainContent(w)
		if err != nil {
			log.Fatal(err)
		}
	}

	w.Run()
}
