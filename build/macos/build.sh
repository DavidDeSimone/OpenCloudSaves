#!/bin/sh 

rm opencloudsaves.dmg || true


cd rclone
go build 
cd ../
mkdir -p bin/
mv rclone/rclone ./bin/

echo "t" > build/macos/macbuildsent.txt
go build
rm build/macos/macbuildsent.txt && touch build/macos/macbuildsent.txt
mkdir -p bin/macos/artifacts
mv opencloudsave bin/macos/artifacts/
cp bin/rclone bin/macos/artifacts/
go run build/macos/bundle.go -bin opencloudsave -icon icon.png -identifier com.opencloudsave.uploads -name "opencloudsaves" -dmg "build/macos/OpenCloudSave.dmg" -o . -assets bin/macos/artifacts/

