#!/usr/bin/env bash

set -e 

rm -rf build/linux/pkg || true
mkdir -p build/linux/pkg/bin
cp -a opencloudsave build/linux/pkg/
cp -a bin/rclone build/linux/pkg/bin/
cd build/linux/
tar -cvz -f opencloudsaves.tar.gz pkg
