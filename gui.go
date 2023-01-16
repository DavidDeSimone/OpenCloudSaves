package main

import (
	"bufio"
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"html/template"
	"os"
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
	Def       *GameDef
	SaveFiles []SaveFile
}

type HtmlInput struct {
	Games []Game
}

func consoleLog(s string) {
	fmt.Println(s)
}

func bindFunctions(w webview.WebView) {
	w.Bind("log", consoleLog)
}

func executeTemplate() (string, error) {
	dm := MakeGameDefManager("")
	games := []Game{}
	for _, v := range dm.GetGameDefMap() {
		game := Game{}
		game.Def = v
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
				info, err := dirFile.Info()
				if err != nil {
					fmt.Println(err)
					continue
				}

				game.SaveFiles = append(game.SaveFiles, SaveFile{
					Filename:   info.Name(),
					ModifiedBy: info.ModTime().Format(time.RFC3339),
					Size:       fmt.Sprintf("%vMB", info.Size()/(1024*1024)),
				})
			}
		}

		games = append(games, game)
	}

	input := HtmlInput{
		Games: games,
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

func GuiMain(ops *Options, dm GameDefManager) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Steam Custom Cloud Uploads")
	w.SetSize(800, 600, webview.HintNone)
	bindFunctions(w)
	html, err := executeTemplate()
	if err != nil {
		fmt.Println(err)
		return
	}

	w.SetHtml(html)
	w.Run()
}
