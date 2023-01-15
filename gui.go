package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/webview/webview"
)

//go:embed html/index.html
var htmlMain string

//go:embed html/style.css
var cssContent string

//go:embed html/main.js
var jsContent string

// @TODO  this is pretty hacky, but I want to keep the executable
// maintained as a single file. We should have this be a build step
// to combine and minfy the finished product
const CSSMarker = "/* ___CSS_AUTO_INJECT___ */"
const JSMarker = "/* __JS_AUTO_INJECT__ */"

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

	jsInject := strings.Replace(htmlMain, JSMarker, jsContent, 1)
	finalHtml := strings.Replace(jsInject, CSSMarker, cssContent, 1)
	w.SetHtml(finalHtml)
	bindFunctions(w)
	w.Run()
}
