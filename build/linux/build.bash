#!/usr/bin/env bash

set -e

cd rclone && go build && cd ../

go build
mv ./opencloudsave ./build/linux/