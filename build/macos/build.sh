#!/bin/sh 

go build
mkdir -p bin/macos/artifacts
mv steamcloudupload bin/macos/artifacts/
go run build/macos/bundle.go -bin steamcloudupload -icon icon.png -identifier com.steamcloud.uploads -name "SteamCloudUploads" -o . -assets bin/macos/artifacts/

