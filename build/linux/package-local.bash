#!/usr/bin/env bash

set -e

flatpak-builder --force-clean --user ./build/linux/build-dir ./build/linux/org.github.opencloudsaves.opencloudsaves.local.yml
