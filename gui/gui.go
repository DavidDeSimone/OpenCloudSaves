package gui

import (
	"bufio"
	"bytes"
	"context"
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
	"strings"
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
	TotalSize      int64
	Def            *core.GameDef
	SaveFiles      []SaveFile
	SaveFilesFound bool
}

type HtmlInput struct {
	Games     []Game
	Platforms []string
	Notice    string
	Version   string
}

func consoleLog(s string) {
	core.InfoLogger.Println(s)
	fmt.Println(s)
}

// @TODO roll this into a class that manages the channel providers
var chanelMutex sync.Mutex
var channelMap map[string]*core.ChannelProvider = make(map[string]*core.ChannelProvider)

func openDirDialog() (string, error) {
	return dialog.Directory().Title("Select Folder").Browse()
}

func cleanupPendingChannel(key string) {
	chanelMutex.Lock()
	_, ok := channelMap[key]
	defer chanelMutex.Unlock()

	if ok {
		delete(channelMap, key)
	}
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
			cleanupPendingChannel(key)
			return "", res.Err
		}

		if res.Finished {
			cleanupPendingChannel(key)
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
	}
	cm := core.MakeCloudManager()

	dm := core.MakeGameDefManager("")
	dm.SetCloudManager(cm)
	ctx, cancel := context.WithCancel(context.Background())
	channels := core.MakeChannelProviderWithCancelFunction(cancel)

	go core.RequestMainOperation(ctx, cm, ops, dm, channels)
	chanelMutex.Lock()
	defer chanelMutex.Unlock()

	channelMap[key] = channels
}

type GuiDatapath struct {
	Path    string
	Include string
}

type GuiGamedef struct {
	Name        string
	Windows     []GuiDatapath
	MacOS       []GuiDatapath
	Linux       []GuiDatapath
	CustomFlags string
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

	if strings.TrimSpace(def.CustomFlags) == "undefined" || strings.TrimSpace(def.CustomFlags) == "" {
		def.CustomFlags = ""
	}

	resultDef := &GuiGamedef{
		Name:        def.DisplayName,
		CustomFlags: def.CustomFlags,
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
	if strings.TrimSpace(gamedef.CustomFlags) == "undefined" || strings.TrimSpace(gamedef.CustomFlags) == "" {
		gamedef.CustomFlags = ""
	}

	dm := core.MakeDefaultGameDefManager()
	gamedefMap := dm.GetGameDefMap()
	gamedefMap[gamedef.Name] = &core.GameDef{
		DisplayName: gamedef.Name,
		SteamId:     "0",
		WinPath:     []*core.Datapath{},
		DarwinPath:  []*core.Datapath{},
		LinuxPath:   []*core.Datapath{},
		CustomFlags: gamedef.CustomFlags,
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

func commitCloudService(ctx context.Context, service int) error {
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
	return cm.CreateDriveIfNotExists(ctx, storage)
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
	}
	ctx, cancel := context.WithCancel(context.Background())
	channels := core.MakeChannelProviderWithCancelFunction(cancel)

	chanelMutex.Lock()
	channelMap[name] = channels
	chanelMutex.Unlock()

	cm := core.MakeCloudManager()
	dm := core.MakeDefaultGameDefManager()
	dm.SetCloudManager(cm)

	go core.RequestMainOperation(ctx, cm, ops, dm, channels)
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
	core.InfoLogger.Println("Commiting Cloud Perfs " + cloudJson)
	cloudperfs := &core.CloudPerfs{}
	err := json.Unmarshal([]byte(cloudJson), cloudperfs)
	if err != nil {
		return err
	}

	err = core.CommitCloudPerfs(cloudperfs)
	if err != nil {
		core.ErrorLogger.Println(err)
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
		obscuredpw, err := cm.ObscurePassword(context.Background(), ftp.Password)
		if err != nil {
			return err
		}

		ftp.Password = obscuredpw
	}

	core.SetFtpDriveStorage(ftp)
	return nil
}

func commitNextCloudSettings(jsonInput string) error {
	nextCloud := &core.NextCloudStorage{}
	err := json.Unmarshal([]byte(jsonInput), nextCloud)
	if err != nil {
		return err
	}

	if nextCloud.Pass != "" {
		cm := core.MakeCloudManager()
		obscuredpw, err := cm.ObscurePassword(context.Background(), nextCloud.Pass)
		if err != nil {
			return err
		}

		nextCloud.Pass = obscuredpw
	}

	core.SetNextCloudStorage(nextCloud)
	return nil
}

func cancelPendingSync(gameName string) {
	core.InfoLogger.Println("Cancel sync of " + gameName)
	chanelMutex.Lock()
	channels, ok := channelMap[gameName]
	chanelMutex.Unlock()

	if ok && channels.Cancel != nil {
		channels.Cancel()
	}
	cleanupPendingChannel(gameName)
}

func deleteCurrentNextCloudSettings() {
	core.DeleteNextCloudStorage(context.Background())
}

func deleteCurrentFTPSettings() {
	core.DeleteFtpDriveStorage(context.Background())
}

func getShouldNotPromptForLargeSyncs() bool {
	cloudperfs := core.GetCurrentCloudPerfsOrDefault()
	return cloudperfs.ShouldNotPromptForLargeSyncs
}

func getMultisyncSelectedGames() (string, error) {
	dm := core.MakeDefaultGameDefManager()
	err := dm.ApplyUserOverrides()
	if err != nil {
		return "", err
	}

	gamedefs := dm.GetGameDefMap()
	result := make(map[string]bool)
	for _, def := range gamedefs {
		if def.SelectInMultisyncMenu {
			result[def.DisplayName] = true
		}
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}

func commitMultisyncSelectGames(gameNames []string) error {
	dm := core.MakeDefaultGameDefManager()
	err := dm.ApplyUserOverrides()
	if err != nil {
		return err
	}

	gamedefs := dm.GetGameDefMap()
	for _, gamename := range gameNames {
		result, ok := gamedefs[gamename]
		if ok {
			result.SelectInMultisyncMenu = true
		}
	}

	return dm.CommitUserOverrides()
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
	w.Bind("deleteCurrentNextCloudSettings", deleteCurrentNextCloudSettings)
	w.Bind("commitNextCloudSettings", commitNextCloudSettings)
	w.Bind("deleteCurrentFTPSettings", deleteCurrentFTPSettings)
	w.Bind("getShouldNotPromptForLargeSyncs", getShouldNotPromptForLargeSyncs)
	w.Bind("cancelPendingSync", cancelPendingSync)
	w.Bind("getMultisyncSelectedGames", getMultisyncSelectedGames)
	w.Bind("commitMultisyncSelectGames", commitMultisyncSelectGames)
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

		var sizeInBytes int64 = 0
		game.SaveFiles = []SaveFile{}
		paths, err := v.GetSyncpaths()
		if err != nil {
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
					continue
				}

				size := info.Size()
				if info.IsDir() {
					size, err = DirSize(datapath.Path + string(os.PathSeparator) + info.Name())
					if err != nil {
						continue
					}
				}

				game.SaveFiles = append(game.SaveFiles, SaveFile{
					Filename:   info.Name(),
					ModifiedBy: info.ModTime().Format(time.RFC3339),
					Size:       fmt.Sprintf("%vMB", size/(1024*1024)),
				})

				sizeInBytes += size
			}
		}

		if len(game.SaveFiles) > 0 {
			game.SaveFilesFound = true
			game.TotalSize = sizeInBytes / (1024 * 1024)
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
		Version:   core.VersionRevision,
	}

	var b bytes.Buffer
	htmlWriter := bufio.NewWriterSize(&b, 8*1024*1024)

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

	multisyncbytes, err := fs.ReadFile(html, "html/multisync.js")
	if err != nil {
		return "", err
	}

	confirmjsbytes, err := fs.ReadFile(html, "html/confirm-modal.js")
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
	settingsjs := fmt.Sprintf("\n<script>%v</script>\n", string(settingsjsbytes))
	confirmjs := fmt.Sprintf("\n<script>%v</script>\n", string(confirmjsbytes))
	multisyncjs := fmt.Sprintf("<script>%v</script>\n", string(multisyncbytes))
	finalResult := css + result + js + syncgamejs + settingsjs + confirmjs + multisyncjs
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
