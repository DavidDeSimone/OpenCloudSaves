#!/bin/sh

set -e

if [ ! -d "libs/webview2" ]; then
    mkdir -p libs/webview2
    curl -sSL "https://www.nuget.org/api/v2/package/Microsoft.Web.WebView2" | /c/Windows/System32/tar.exe -xf - -C libs/webview2
    cp libs/webview2/build/native/x64/WebView2Loader.dll build/win
fi

export CGO_CXXFLAGS="-I$(pwd)/libs/webview2/build/native/include"

go build -ldflags="-H windowsgui" && cp -a steamcloudupload.exe build/win/