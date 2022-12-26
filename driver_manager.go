package main

import "log"

type DriverManager struct {
	drivers map[string]Driver
}

func (d *DriverManager) LoadDefaultGameMap() {
	// TODO
}

func (d *DriverManager) GetFilesForGame(id string) ([]string, error) {
	driver, ok := d.drivers[id]
	if !ok {
		log.Fatalf("Failed to find game (%v)", id)
	}
	return driver.GetFilenames()
}

// This is to poss. hook up different platforms having different names for the games.
// It would be better to move to steamid
func (d *DriverManager) GetCanonicalNameForGame(id string) string {
	return id
}

type Driver interface {
	GetFilenames() ([]string, error)
}
