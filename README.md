# Custom Steam Cloud Uploads
This project is a cross platform, open source executable to allow cross platform saves across games. 



## Build

### Windows

For windows builds, you will need [MSYS2](https://www.msys2.org/) or a stand alone version of GCC. 

You will also need to download Webview 2 - https://developer.microsoft.com/en-us/microsoft-edge/webview2/#download-section

I've only tested this on MSYS2 MSYS (purple icon).

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

From there, you can run:
```bash
go get
go build
./steamcloudupload.exe
```


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
steamcloudupload[EXT] <FLAGS>
```

You can also run 
```bash
go run *.go --verbose
```

### Linux

In addition to golang, you will need the following deps depending on your distro:

Debian / Ubuntu: 
```
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev libgtk-3-dev
```
