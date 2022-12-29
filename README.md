# Custom Steam Cloud Uploads
This project is a cross platform, open source executable to allow cross platform saves across games. 

## Build

### Windows

For windows builds, you will need [MSYS2](https://www.msys2.org/)

I've only tested this on MSYS2 MSYS (purple icon) - I have not gotten this working on MSYS2-URT. 

Run
```bash
pacman -S mingw-w64-x86_64-go
export GOROOT=/mingw64/lib/go
export GOPATH=/mingw64
```

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
./steamcloudupload.exe --gui
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
go run *.go --gui --verbose
```