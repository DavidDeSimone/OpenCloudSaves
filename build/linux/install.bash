#!/bin/bash

if ! (zenity --question --title="Disclaimer" --text="This software is in a open beta. If you have valuable save data, it is recommended that you make a manual backup. OpenCloudSave is not responsible for lost data. Do you agree and wish to continue?" --width=600 2> /dev/null); then
    zenity --error --title="Terms Denied" --text="Terms were denied, cannot proceed." --width=300 2> /dev/null
    exit 1
fi
hasPass=$(passwd -S "$USER" | awk -F " " '{print $2}')
if [[ $hasPass != "P" ]]; then
    zenity --error --title="Password Error" --text="Password is not set, please set one in the terminal with the <b>passwd</b> command, then run this again." --width=400 2> /dev/null
    exit 1
fi
PASSWD="$(zenity --password --title="Enter Password" --text="Enter Deck User Password (not Steam account!)" 2>/dev/null)"
echo "$PASSWD" | sudo -v -S
ans=$?
if [[ $ans == 1 ]]; then
    zenity --error --title="Password Error" --text="Incorrect password provided, please run this command again and provide the correct password." --width=400 2> /dev/null
    exit 1
fi
mkdir -p /tmp/
curl https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.16.0/linux_opencloudsaves_0.16.0_x86_64.flatpak > /tmp/opencloudsaves.flatpak
flatpak install --noninteractive --or-update /tmp/opencloudsaves.flatpak
curl https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.16.0/OpenCloudSave.desktop > ~/Desktop/OpenCloudSave.desktop
#curl desktop file, put on desktop