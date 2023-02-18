#!/usr/bin/env bash

set -e

if ! (zenity --question --title="Welcome" --text="This software is in a open beta. If you have valuable save data, it is recommended that you make a manual backup. OpenCloudSave is not responsible for lost data. Do you agree and wish to continue?" --width=500 2> /dev/null); then
    zenity --error --title="Usaged Denied" --text="Not proceeding further with install. Run this script again if you change your mind. Thank you." --width=500 2> /dev/null
    exit 1
fi

passIsSet=$(passwd -S "$USER" | awk -F " " '{print $2}')
if [[ $passIsSet != "P" ]]; then
    zenity --error --title="Password Error" --text="Password is not set, please set one in the terminal with the <b>passwd</b> command, then run this again." --width=400 2> /dev/null
    exit 1
fi

if [[ -d ~/Desktop/ ]]; then
    curl -L "https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.16.0/OpenCloudSave.desktop" > ~/Desktop/OpenCloudSave.desktop
else 
    echo 'Skipping desktop file installation - Desktop not found.'
fi
