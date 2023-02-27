#!/bin/bash

set -e

if ! (zenity --question --title="Welcome" --text="This software is in a open beta. If you have valuable save data, it is recommended that you make a manual backup. OpenCloudSave is not responsible for lost data. Do you agree and wish to continue?" --width=500 2> /dev/null); then
    zenity --error --title="Usaged Denied" --text="Not proceeding further with install. Run this script again if you change your mind. Thank you." --width=500 2> /dev/null
    exit 1
fi

mkdir -p /tmp/
curl -L "https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.17.4/linux_opencloudsaves_0.17.4_x86_64.flatpak" > /tmp/opencloudsaves.flatpak
flatpak install --user --noninteractive --or-update /tmp/opencloudsaves.flatpak

if [[ -d ~/Desktop/ ]]; then
    curl -L "https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.17.4/OpenCloudSave.desktop" > ~/Desktop/OpenCloudSave.desktop
elif [[ -d /home/deck/Desktop/ ]]; then
    curl -L "https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.17.4/OpenCloudSave.desktop" > /home/deck/Desktop/OpenCloudSave.desktop
else 
    zenity --error --title="Cannot find desktop location" --text="The install script cannot find your desktop. Please report this as a bug to https://github.com/DavidDeSimone/OpenCloudSaves/" -- width=500 2> /dev/null
fi
