#!/usr/bin/bash

set -e

CURR=$(pwd)
cd build/win/

"/c/Program Files (x86)/WiX Toolset v3.11/bin/candle.exe" opencloudsave.wxs
"/c/Program Files (x86)/WiX Toolset v3.11/bin/light.exe" -ext WixUIExtension -cultures:en-us opencloudsave.wixobj

cd $CURR