package gui

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"opencloudsave/core"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sqweek/dialog"
	"github.com/webview/webview"
)

//go:embed html
var html embed.FS

//go:embed NOTICE.txt
var notice string

type SaveFile struct {
	Filename   string
	ModifiedBy string
	Size       string
}

type Game struct {
	Name           string
	Def            *core.GameDef
	SaveFiles      []SaveFile
	SaveFilesFound bool
}

type HtmlInput struct {
	Games     []Game
	Platforms []string
	Notice    string
}

func consoleLog(s string) {
	fmt.Println(s)
}

var chanelMutex sync.Mutex
var channelMap map[string]*core.ChannelProvider = make(map[string]*core.ChannelProvider)

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
	case res := <-channels.Logs:
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

func syncGame(key string) {
	ops := &core.Options{
		Gamenames: []string{key},
		Verbose:   []bool{true},
	}
	cm := core.MakeCloudManager()

	dm := core.MakeGameDefManager("")
	dm.SetCloudManager(cm)
	channels := &core.ChannelProvider{
		Logs: make(chan core.Message, 100),
	}

	go core.RequestMainOperation(cm, ops, dm, channels)
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

// @TODO the issue with this is that when we refresh, we will
// rebuild the list. Instead, we may need a 'hidden' flag and not
// actually delete entries.
func removeGamedefByKey(key string) {
	dm := core.MakeDefaultGameDefManager()
	dm.ApplyUserOverrides()
	dm.RemoveGameDef(key)
	dm.CommitUserOverrides()
}

func fetchGamedef(key string) (*GuiGamedef, error) {
	dm := core.MakeDefaultGameDefManager()
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
	dm := core.MakeDefaultGameDefManager()
	gamedefMap := dm.GetGameDefMap()
	gamedefMap[gamedef.Name] = &core.GameDef{
		DisplayName: gamedef.Name,
		SteamId:     "0",
		WinPath:     []*core.Datapath{},
		DarwinPath:  []*core.Datapath{},
		LinuxPath:   []*core.Datapath{},
	}

	for _, def := range gamedef.Windows {
		gamedefMap[gamedef.Name].WinPath = append(gamedefMap[gamedef.Name].WinPath, &core.Datapath{
			Path:    def.Path,
			Include: def.Include,
		})
	}

	for _, def := range gamedef.MacOS {
		gamedefMap[gamedef.Name].DarwinPath = append(gamedefMap[gamedef.Name].DarwinPath, &core.Datapath{
			Path:    def.Path,
			Include: def.Include,
		})
	}

	for _, def := range gamedef.Linux {
		gamedefMap[gamedef.Name].LinuxPath = append(gamedefMap[gamedef.Name].LinuxPath, &core.Datapath{
			Path:    def.Path,
			Include: def.Include,
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
	cloudperfs := core.GetCurrentCloudPerfsOrDefault()
	cloudperfs.Cloud = service
	err := core.CommitCloudPerfs(cloudperfs)
	if err != nil {
		return err
	}

	storage, err := core.GetCurrentCloudStorage()
	if err != nil {
		return err
	}

	cm := core.MakeCloudManager()
	return cm.CreateDriveIfNotExists(storage)
}

func getCloudService() (int, error) {
	cloudperfs, err := core.GetCurrentCloudPerfs()
	if err != nil {
		return -1, nil
	}

	return cloudperfs.Cloud, nil
}

func getSyncDryRun(name string) error {
	ops := &core.Options{
		DryRun:    []bool{true},
		Gamenames: []string{name},
		Verbose:   []bool{true},
	}
	channels := &core.ChannelProvider{
		Logs: make(chan core.Message, 100),
	}

	chanelMutex.Lock()
	channelMap[name] = channels
	chanelMutex.Unlock()

	cm := core.MakeCloudManager()
	dm := core.MakeDefaultGameDefManager()
	dm.SetCloudManager(cm)

	go core.RequestMainOperation(cm, ops, dm, channels)
	return nil
}

func getShouldPerformDryRun() (bool, error) {
	cloudperfs := core.GetCurrentCloudPerfsOrDefault()
	return cloudperfs.PerformDryRun, nil
}

func getCloudPerfs() (string, error) {
	cloudperfs := core.GetCurrentCloudPerfsOrDefault()
	json, err := json.Marshal(cloudperfs)
	return string(json), err
}

func commitCloudPerfs(cloudJson string) error {
	fmt.Println("Commiting Cloud Perfs " + cloudJson)
	cloudperfs := &core.CloudPerfs{}
	err := json.Unmarshal([]byte(cloudJson), cloudperfs)
	if err != nil {
		return err
	}
	fmt.Println("Writing cloud perfs")
	err = core.CommitCloudPerfs(cloudperfs)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func clearUserSettings() error {
	path := core.GetDefaultUserOverridePath()
	return os.Remove(path)
}

func commitFTPSettings(jsonInput string) error {
	ftp := &core.FtpStorage{}
	err := json.Unmarshal([]byte(jsonInput), ftp)
	if err != nil {
		return err
	}

	if ftp.Password != "" {
		cm := core.MakeCloudManager()
		obscuredpw, err := cm.ObscurePassword(ftp.Password)
		if err != nil {
			return err
		}

		ftp.Password = obscuredpw
	}

	core.SetFtpDriveStorage(ftp)
	return nil
}

func deleteCurrentFTPSettings() {
	core.DeleteFtpDriveStorage()
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
	w.Bind("getCloudPerfs", getCloudPerfs)
	w.Bind("commitCloudPerfs", commitCloudPerfs)
	w.Bind("clearUserSettings", clearUserSettings)
	w.Bind("commitFTPSettings", commitFTPSettings)
	w.Bind("deleteCurrentFTPSettings", deleteCurrentFTPSettings)
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

func buildGamelist(dm core.GameDefManager) []Game {
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
				if datapath.Include != "" {
					match, err := filepath.Match(datapath.Include, dirFile.Name())
					if !match || err != nil {
						continue
					}
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
	dm := core.MakeDefaultGameDefManager()
	games := buildGamelist(dm)

	sort.Slice(games, func(i, j int) bool {
		return games[i].Def.DisplayName < games[j].Def.DisplayName
	})
	input := HtmlInput{
		Games:     games,
		Platforms: []string{"Windows", "MacOS", "Linux"},
		Notice:    notice,
	}

	var b bytes.Buffer
	htmlWriter := bufio.NewWriterSize(&b, 4*1024*1024)

	templ := template.Must(template.ParseFS(html, "html/index.html"))
	err := templ.Execute(htmlWriter, input)
	if err != nil {
		return "", err
	}

	syncgamejsbytes, err := fs.ReadFile(html, "html/syncgame.js")
	if err != nil {
		return "", err
	}

	settingsjsbytes, err := fs.ReadFile(html, "html/settings.js")
	if err != nil {
		return "", err
	}

	jsContent, err := fs.ReadFile(html, "html/main.js")
	if err != nil {
		return "", err
	}

	cssContent, err := fs.ReadFile(html, "html/style.css")
	if err != nil {
		return "", err
	}

	htmlWriter.Flush()
	result := b.String()
	js := fmt.Sprintf("<script>%v</script>", string(jsContent))
	css := fmt.Sprintf("<style>%v</style>", string(cssContent))
	syncgamejs := fmt.Sprintf("\n<script>%v</script>\n", string(syncgamejsbytes))
	settingsjs := fmt.Sprintf("<script>%v</script>\n", string(settingsjsbytes))
	finalResult := css + result + js + syncgamejs + settingsjs
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

func GuiMain(ops *core.Options, dm core.GameDefManager) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Open Cloud Save")
	w.SetSize(800, 600, 0)
	bindFunctions(w)

	storage := core.GetCurrentStorageProvider()
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
