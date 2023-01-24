#!/usr/bin/bash

set -e

## TODO update to include rclone.exe

CURR=$(pwd)
cd build/win/

"/c/Program Files (x86)/WiX Toolset v3.11/bin/candle.exe" product.wxs
"/c/Program Files (x86)/WiX Toolset v3.11/bin/light.exe" product.wixobj

cd $CURR