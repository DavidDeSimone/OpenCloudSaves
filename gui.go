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

func syncGame(key string) {
	ops := &Options{
		Gamenames: []string{key},
	}
	dm := MakeGameDefManager("")
	channels := &ChannelProvider{
		logs:   make(chan Message, 100),
		cancel: make(chan Cancellation, 1),
		input:  make(chan SyncRequest, 10),
		output: make(chan SyncResponse, 10),
	}

	go consoleLogger(channels.logs)
	go CliMain(ops, dm, channels, SyncOp)
}

func bindFunctions(w webview.WebView) {
	w.Bind("log", consoleLog)
	w.Bind("syncGame", syncGame)
	w.Bind("refresh", func() {
		refreshMainContent(w)
	})
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
			dirFiles, err := os.ReadDir(datapath.Path)
			if err != nil {
				fmt.Println(err)
				continue
			}

			for _, dirFile := range dirFiles {
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
