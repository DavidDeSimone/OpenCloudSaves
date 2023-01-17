Install MSYS2
Use MSYS2 MINGW64 

in CMD line run
```
mkdir libs\webview2
curl -sSL "https://www.nuget.org/api/v2/package/Microsoft.Web.WebView2" | tar -xf - -C libs\webview2
copy /Y libs\webview2\build\native\x64\WebView2Loader.dll build
```

Specify the absoulte path to your header files

run 

```
go build -ldflags="-H windowsgui"
```

The WebView2Loader.dll needs to be in the same DIR as the EXE