package main

import (
	"bufio"
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"html/template"
	"time"

	"github.com/webview/webview"
)

//go:embed html/index.html
var htmlMain embed.FS

//go:embed html/style.css
var cssContent string

//go:embed html/main.js
var jsContent string

type HtmlInput struct {
}

func consoleLog(s string) {
	fmt.Println(s)
}

func bindFunctions(w webview.WebView) {
	w.Bind("log", consoleLog)
}

func executeTemplate() string {
	input := HtmlInput{}

	var b bytes.Buffer
	htmlWriter := bufio.NewWriter(&b)

	templ := template.Must(template.ParseFS(htmlMain, "html/index.html"))
	templ.Execute(htmlWriter, input)

	result := b.String()
	js := fmt.Sprintf("<script>%v</script>", jsContent)
	css := fmt.Sprintf("<style>%v</style>", cssContent)
	return css + result + js
}

func GuiMain(ops *Options, dm GameDefManager) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Steam Custom Cloud Uploads")
	w.SetSize(800, 600, webview.HintNone)
	fmt.Println(time.Now().UnixMilli())
	bindFunctions(w)
	w.SetHtml(executeTemplate())
	fmt.Println(time.Now().UnixMilli())
	w.Run()
}
