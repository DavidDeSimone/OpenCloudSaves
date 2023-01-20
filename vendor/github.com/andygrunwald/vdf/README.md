# vdf: A Lexer and Parser for Valves Data Format (known as vdf)

[![GoDoc](https://pkg.go.dev/badge/github.com/andygrunwald/vdf?utm_source=godoc)](https://pkg.go.dev/github.com/andygrunwald/vdf)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://goreportcard.com/report/github.com/andygrunwald/vdf)

A Lexer and Parser for [Valves Data Format (known as vdf)](https://developer.valvesoftware.com/wiki/KeyValues) written in Go.

## Installation

It is go gettable

```
$ go get github.com/andygrunwald/vdf
```

## Usage

Given a file named [`gamestate_integration_consolesample.cfg`](testdata/gamestate_integration_consolesample.cfg) with content:

```
"Console Sample v.1"
{
	"uri" 		"http://127.0.0.1:3000"
	"timeout" 	"5.0"
	"buffer"  	"0.1"
	"throttle" 	"0.5"
	"heartbeat"	"60.0"
	[...]
}
```

Can be parsed with this Go code:

```go
package main

import (
	"fmt"
	"os"

	"github.com/andygrunwald/vdf"
)

func main() {
	f, err := os.Open("gamestate_integration_consolesample.cfg")
	if err != nil {
		panic(err)
	}

	p := vdf.NewParser(f)
	m, err := p.Parse()
	if err != nil {
		panic(err)
	}

	fmt.Println(m)
}
```

And it will output:

```
map[
	Console Sample v.1:map[
		uri:http://127.0.0.1:3000
		timeout:5.0
		buffer:0.1
		throttle:0.5
		heartbeat:60.0
		[...]
	]
]
```

## API-Documentation

The official Go package documentation can be found at https://pkg.go.dev/github.com/andygrunwald/vdf.

## Development

### Unit testing

To run the local unit tests, execute

```sh
$ make test
```

To run the local unit tests and view the unit test code coverage in your local web browser, execute

```sh
$ make test-coverage-html
```

### Fuzzing tests

This library implements [Go fuzzing](https://go.dev/security/fuzz/).
The generated fuzzing corpus is stored in [andygrunwald/vdf-fuzzing-corpus](https://github.com/andygrunwald/vdf-fuzzing-corpus/), to avoid blowing up the size of this repository.

To run fuzzing locally, execute

```sh
$ make init-fuzzing   # Clone the corpus into testdata/fuzz
$ make clean-fuzzing  # Clean the local fuzzing cache
$ make test-fuzzing   # Execute the fuzzing
```

## VDF parser in other languages

* PHP and JavaScript: [rossengeorgiev/vdf-parser](https://github.com/rossengeorgiev/vdf-parser)
* PHP: [devinwl/keyvalues-php](https://github.com/devinwl/keyvalues-php)
* PHP: [lukezbihlyj/vdf-parser](https://github.com/lukezbihlyj/vdf-parser)
* PHP: [EXayer/vdf-converter](https://github.com/EXayer/vdf-converter)
* C#: [sanmadjack/VDF](https://github.com/sanmadjack/VDF)
* C#: [shravan2x/Gameloop.Vdf](https://github.com/shravan2x/Gameloop.Vdf)
* C#: [Indieteur/Steam-Apps-Management-API](https://github.com/Indieteur/Steam-Apps-Management-API)
* C#: [GerhardvanZyl/Steam-VDF-Converter](https://github.com/GerhardvanZyl/Steam-VDF-Converter)
* C++: [TinyTinni/ValveFileVDF](https://github.com/TinyTinni/ValveFileVDF)
* Java: [DHager/hl2parse](https://github.com/DHager/hl2parse)
* JavaScript [node-steam/vdf](https://github.com/node-steam/vdf)
* JavaScript: [Corecii/steam-binary-vdf-ts](https://github.com/Corecii/steam-binary-vdf-ts)
* JavaScript: [RoyalBingBong/vdfplus](https://github.com/RoyalBingBong/vdfplus)
* JavaScript: [key-values/key-values-ts](https://github.com/key-values/key-values-ts)
* Python: [ValvePython/vdf](https://github.com/ValvePython/vdf)
* Python: [gorgitko/valve-keyvalues-python](https://github.com/gorgitko/valve-keyvalues-python)
* Python: [noriah/PyVDF](https://github.com/noriah/PyVDF)
* Go: [marshauf/keyvalues](https://github.com/marshauf/keyvalues)
* Go: [Jleagle/steam-go](https://github.com/Jleagle/steam-go/)
* Go: [Wakeful-Cloud/vdf](https://github.com/Wakeful-Cloud/vdf)
* Rust: [LovecraftianHorror/vdf-rs](https://github.com/LovecraftianHorror/vdf-rs)
* Rust: [Corecii/steam_vdf](https://github.com/Corecii/steam_vdf)
* Other (VS-Code Extension): [cooolbros/vscode-vdf](https://github.com/cooolbros/vscode-vdf)
* And some more: [Github search for vdf valve](https://github.com/search?p=1&q=vdf+valve&ref=searchresults&type=Repositories&utf8=%E2%9C%93)

## Inspiration

The code is inspired by [@benbjohnson](https://github.com/benbjohnson)'s article [Handwritten Parsers & Lexers in Go](https://blog.gopheracademy.com/advent-2014/parsers-lexers/) and his example [sql-parser](https://github.com/benbjohnson/sql-parser).
Thank you Ben!

## License

This project is released under the terms of the [MIT license](http://en.wikipedia.org/wiki/MIT_License).
