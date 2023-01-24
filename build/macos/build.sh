#!/bin/sh 

cd rclone
go build 
cd ../
mkdir -p bin/
mv rclone/rclone ./bin/

go build
mkdir -p bin/macos/artifacts
mv opencloudsave bin/macos/artifacts/
cp bin/rclone bin/macos/artifacts/
go run build/macos/bundle.go -bin opencloudsave -icon icon.png -identifier com.opencloudsave.uploads -name "opencloudsaves" -o . -assets bin/macos/artifacts/

