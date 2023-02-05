#!/usr/bin/env bash

set -e

# Requied due to https://vielmetti.typepad.com/logbook/2022/10/git-security-fixes-lead-to-fatal-transport-file-not-allowed-error-in-ci-systems-cve-2022-39253.html
git config protocol.file.allow always
flatpak-builder --force-clean --user --install --repo=repo ./build/linux/build-dir ./build/linux/org.github.opencloudsaves.opencloudsaves.local.yml
flatpak build-bundle repo opencloudsave.flatpak io.github.daviddesimone.opencloudsaves