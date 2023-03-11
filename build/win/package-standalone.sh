#!/usr/bin/bash

set -e

CURR=$(pwd)
cd build/win/

if [[ ! -f MicrosoftEdgeWebView2RuntimeInstallerX64.exe ]]; then
    echo "You will need to download the standalone evergreen install from https://developer.microsoft.com/en-us/microsoft-edge/webview2/ and place that .exe in build/win" 
    exit 1
fi

"/c/Program Files (x86)/WiX Toolset v3.11/bin/candle.exe" opencloudsave-standalone.wxs
"/c/Program Files (x86)/WiX Toolset v3.11/bin/light.exe" -ext WixUIExtension -ext WixUtilExtension -cultures:en-us opencloudsave-standalone.wixobj

cd $CURR