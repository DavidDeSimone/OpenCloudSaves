#!/usr/bin/env bash

set -e

flatpak-builder --force-clean --user ./build/linux/build-dir ./build/linux/org.github.opencloudsaves.opencloudsaves.yml
flatpak build-bundle repo opencloudsave.flatpak io.github.daviddesimone.opencloudsaves