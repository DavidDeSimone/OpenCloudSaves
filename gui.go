package main

import (
	_ "embed"
	"fmt"

	"github.com/webview/webview"
)

//go:embed html/index.html
var htmlMain string

//go:embed html/addgame.html
var htmlAddGame string

func consoleLog(s string) {
	fmt.Println(s)
}

func bindFunctions(w webview.WebView) {
	w.Bind("log", consoleLog)
}

func GuiMain(ops *Options, dm GameDefManager) {
	debug := true
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("Steam Custom Cloud Uploads")
	w.SetSize(800, 600, webview.HintNone)
	w.SetHtml(htmlAddGame)
	bindFunctions(w)
	w.Run()
}
