#!/bin/sh

set -e

echo 'Building....'

cd rclone
go build
cd ../
mkdir -p ./bin/
mv ./rclone/rclone.exe ./bin/

if [ ! -d "libs/webview2" ]; then
    mkdir -p libs/webview2
    curl -sSL "https://www.nuget.org/api/v2/package/Microsoft.Web.WebView2" | /c/Windows/System32/tar.exe -xf - -C libs/webview2
    cp libs/webview2/build/native/x64/WebView2Loader.dll build/win
fi

export CGO_CXXFLAGS="-I$(pwd)/libs/webview2/build/native/include"


if ! command -v go-winres &> /dev/null
then
    go install github.com/tc-hib/go-winres@v0.3.1
fi

go-winres make

go build -ldflags="-H windowsgui" -tags=$1 && mv opencloudsave.exe build/win/

echo 'Build Complete....'