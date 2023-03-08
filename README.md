# Open Cloud Saves 

Join us over at [Our Discord](https://discord.gg/BPQ3peQxXA)

Open Cloud Saves is an open source application for managing your saves games across Windows, MacOS, and Linux (including SteamOS). Open Cloud Saves is available for use officially as a “beta”. As a beta test, we recommend that you manually make a backup of your save data before usage. Until Open Cloud Save is more battle tested, we will issue a warning for users to use caution with “critical, beloved” save data.

Open Cloud Save gives an advantage over existing cloud solutions:

1. Allows cloud saves for games without developer support
2. Allows for the exclusion of certain files or filetypes. This can prevent games syncing graphical settings in addition to syncing save data.
3. Allows for sync between storefronts (you own a Steam on linux and a Epic Game Store version on windows

Key Features:

1. Inclusion of specific save files based on [pattern matching](https://rclone.org/filtering/)
2. Customizable save data locations - you can tailor the app to your specific save locations
3. Ability to create new save definitions - you do not need to wait for developers to support cloud saves for their games.
4. Data protection - by default, OpenCloudSave will perform a dry-run before all syncs. This way you can see what changes will be made to your save data before they happen. (You can disable this functionality if you just want to immediately sync)

<p align="center">

![image](https://user-images.githubusercontent.com/7245174/218942321-510179b1-1f18-4ea6-8e91-6cbabae63672.png)

</p>


# Install

## Linux / Steam Deck

1. We are now listed on [Flathub](https://flathub.org/apps/details/io.github.daviddesimone.opencloudsaves) and the Discover store on steam deck. We recommend downloading through flathub or the discover store on steam deck!

For power users we offer a precompiled binary for [x86_64 in .tar.gz format](https://github.com/DavidDeSimone/OpenCloudSaves/releases/tag/v0.17.8.0). 

## Windows

1. Download the [Open Cloud Save Installer](https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.17.8.0/windows_opencloudsave_0.17.8_x86_64.msi)
2. Follow the instructions for installation.
3. Launch opencloudsave.exe located in C:\Program Files\OpenCloudSave\

NOTE: There is a chance that windows defender or your AV may flag this application as a "virus". This is a consequence of the application being written in golang. Windows marks a lot of golang applications as viruses, as documented here: 
https://go.dev/doc/faq#virus

This application is free and open source, and is free to audit.

## MacOS
1. Download the [Open Cloud Save .dmg](https://github.com/DavidDeSimone/OpenCloudSaves/releases/download/v0.17.8.0/macos_opencloudsaves_0.17.8_aarch64.dmg)
2. Drag the executable into your /Applications/ directory
3. Launch OpenCloudSave


## Build

On all platforms, you will need to initialize the rclone submodule:

```
git submodule update --init
```

### Windows

For windows builds, you will need [MSYS2](https://www.msys2.org/). Specifically, you will need MINGW64 

You will also need to download Webview 2 - https://developer.microsoft.com/en-us/microsoft-edge/webview2/#download-section. This is a runtime requirement that is usually downloaded by our MSI installer. If you haven't already installed OpenCloudSave, you will need to download WebView2 to run your compiled application. 

Run
```bash
pacman -S mingw-w64-x86_64-go
PATH=$PATH:/mingw64/bin/
export GOROOT=/mingw64/lib/go
export GOPATH=/mingw64
```

You will likely want to add these to your shell start up script.

Validate that go is installed correctly with
```bash
go version
```

You will also need gcc:
```bash
pacman -S mingw-w64-x86_64-gcc
```


NOTE: Running our build script will install go-winres. go-winres is a tool that will compile things like our icon into our application. You can inspect the code at https://github.com/tc-hib/

From there, you can run:
```bash
./build/win/build.sh
./build/win/opencloudsave.exe
```

If you would like to view the console logs of the application, you can build via:

```bash
./build/win/build.sh debug
```

To build the MSI, you will need WiX toolset v3.11. https://wixtoolset.org/

Once installed, run:

```bash
./build/win/package.sh
```

This will create the MSI in the build/win directory.

### MacOS

For this build, you will need golang and a C compiler. 

You may need to install xcode command line tools if you haven't already:

`xcode-select --install`

To build, you will need to run 
```bash
go build
```

From there, you can run 
```bash
./opencloudsave
```

To package the finished application, you can run
```bash
./build/macos/build.sh
```

This will compile a macOS application (as opencloudsave.app) and a DMG for distribution.

### Linux

We support both a direct golang build, and a flatpak build

#### Local Build

In addition to golang, you will need the following deps depending on your distro:

Debian / Ubuntu: 
```
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libgtk-3-dev webkit2gtk-4.0
```

From there you can run 
```bash
go build
./opencloudsave
```

#### Flatpak

For this example, you will need [flatpak](https://flatpak.org/setup/). This assumes you have basic familiarity with flatpak, but if  you do not, you should be up to speed after running flatpak's getting started. 

You can run 
```bash
./build/linux/package-local.bash
```

This will build the flatpak - from there you can install it and run the flatpak version. `package-local` builds off of the current state of the repo, so you can iterate and build the app without needing to install any deps beside flatpak and flatpak-builder.


The following command builds the "release" version of the app. This is tied to a specific commit to ensure that the build is reproducible. 
```bash
./build/linux/package.bash
```
