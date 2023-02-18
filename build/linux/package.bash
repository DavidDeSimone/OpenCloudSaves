#!/usr/bin/env bash

set -e

flatpak-builder --force-clean --user ./build/linux/build-dir ./io.github.daviddesimone.opencloudsaves.yml
flatpak build-bundle repo opencloudsave.flatpak io.github.daviddesimone.opencloudsaves