#!/bin/bash

set -e 

rm core/version.txt || true
git describe --tags --abbrev=0 >> core/version.txt
git rev-parse HEAD >> core/version.txt