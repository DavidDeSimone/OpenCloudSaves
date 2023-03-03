#!/bin/sh 

rm opencloudsaves.dmg || true

if [ ! -f "build/macos/OpenCloudSave.dmg" ]; then
    curl "https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/macos-dmg-v1.0.0/OpenCloudSave-release.dmg" > build/macos/OpenCloudSave.dmg
fi

./build/gen-version-rev.bash

cd rclone
go build 
cd ../
mkdir -p bin/
mv rclone/rclone ./bin/

go build
mkdir -p bin/macos/artifacts
mv opencloudsave bin/macos/artifacts/
cp bin/rclone bin/macos/artifacts/
go run build/macos/bundle.go -bin opencloudsave -icon icon.png -identifier com.opencloudsave.uploads -name "opencloudsaves" -dmg "build/macos/OpenCloudSave.dmg" -o . -assets bin/macos/artifacts/

