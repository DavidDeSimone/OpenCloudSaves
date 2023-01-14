# Custom Steam Cloud Uploads
This project is a cross platform, open source executable to allow cross platform saves across games. 

## Build

### Windows

For windows builds, you will need [MSYS2](https://www.msys2.org/)

I've only tested this on MSYS2 MSYS (purple icon) - I have not gotten this working on MSYS2-URT. 

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
<!-- 
Fedora: 
```
sudo dnf install golang gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel 
```

Arch Linux: 
```
sudo pacman -S go xorg-server-devel libxcursor libxrandr libxinerama libxi
```

Solus: 
```
sudo eopkg it -c system.devel golang mesalib-devel libxrandr-devel libxcursor-devel libxi-devel libxinerama-devel
```

openSUSE: 
```
sudo zypper install go gcc libXcursor-devel libXrandr-devel Mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
```

Void Linux: 
```
sudo xbps-install -S go base-devel xorg-server-devel libXrandr-devel libXcursor-devel libXinerama-devel
```

Alpine Linux 
```
sudo apk add go gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev
``` -->
