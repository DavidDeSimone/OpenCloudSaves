#!/bin/sh 

go build
mkdir -p bin/macos/artifacts
mv opencloudsave bin/macos/artifacts/
go run build/macos/bundle.go -bin opencloudsave -icon icon.png -identifier com.steamcloud.uploads -name "opencloudsaves" -o . -assets bin/macos/artifacts/

