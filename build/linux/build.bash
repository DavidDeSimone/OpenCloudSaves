#!/usr/bin/env bash

set -e

cd rclone && go build && cd ../
mkdir -p bin/
cp rclone/rclone ./bin

go build
mv ./opencloudsave ./build/linux/