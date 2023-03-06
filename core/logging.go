package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var InfoLogger *log.Logger
var ErrorLogger *log.Logger

const DefaultLogPath = "opencloudsave.log"

func InitLoggingWithDefaultPath() error {
	path, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	return InitLoggingWithPath(filepath.Join(path, DefaultLogPath))
}

func InitLoggingWithPath(path string) error {
	fmt.Println("Creating logfile at " + path)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	// defer file.Close()
	InfoLogger = log.New(file, "INFO\t", log.Ldate|log.Ltime)
	ErrorLogger = log.New(file, "ERROR\t", log.Lshortfile|log.Ldate|log.Ltime)
	log.SetOutput(file)
	return nil
}
