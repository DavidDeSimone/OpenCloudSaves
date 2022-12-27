package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
)

//go:embed driver_map.json
var driverMap []byte

type Driver struct {
	win_path   string
	linux_path string
	macos_path string

	saves_cross_compatible bool
	save_ext               string
}

func (d *Driver) GetFilenames() ([]string, error) {
	return nil, nil
}

func (d *Driver) GetSyncpath() (string, error) {
	return "", nil
}

type DriverManager struct {
	drivers map[string]Driver
}

func MakeDriverManager() *DriverManager {
	dm := &DriverManager{}
	fmt.Println(string(driverMap))
	json.Unmarshal(driverMap, dm)
	return dm
}

func (d *DriverManager) GetFilesForGame(id string) ([]string, error) {
	driver, ok := d.drivers[id]
	if !ok {
		log.Fatalf("Failed to find game (%v)", id)
	}
	return driver.GetFilenames()
}

func (d *DriverManager) GetSyncpathForGame(id string) (string, error) {
	driver, ok := d.drivers[id]
	if !ok {
		log.Fatalf("Failed to find game (%v)", id)
	}

	return driver.GetSyncpath()
}

// This is to poss. hook up different platforms having different names for the games.
// It would be better to move to steamid
func (d *DriverManager) GetCanonicalNameForGame(id string) string {
	return id
}

// type Driver interface {
// 	GetFilenames() ([]string, error)
// 	GetSyncpath() (string, error)
// }
