# Open Cloud Saves 
This project is a cross platform, open source executable to allow cross platform saves across games. This project is currently in beta and requires a invitation to use.

<p align="center"><img width="791" alt="Screenshot 2023-01-16 at 9 30 34 PM" src="https://user-images.githubusercontent.com/7245174/212817944-2f9ff2cc-c03a-4b1f-a50c-309d3e36f3e1.png"></p>

## Build

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
go get && go build
```

From there, you can run 
```bash
opencloudsave[EXT] <FLAGS>
```

To package the finished application, you can run
```bash
./build/macos
```

This will compile an application.

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


The following command builds the "release" version of the app. This is tied to a specific commit to ensure that the build is reproducable. 
```bash
./build/linux/package.bash
```

## Tests

You can run integration tests with 
```bash
go test
```

Note: These tests modify your local file system in the same way that OpenCloudSave modifies your system. It will write to your tempdir and cache dir.