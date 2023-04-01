package gui

import "github.com/webview/webview"

var window webview.WebView

func SetRootWindow(w webview.WebView) {
	window = w
}

func GetRootWindow() webview.WebView {
	return window
}

func DestroyRootWindow() {
	window.Destroy()
}
