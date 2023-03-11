package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
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
	logger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    250, // Megabytes
		MaxAge:     30,  // Days
		MaxBackups: 1,
	}

	InfoLogger = log.New(logger, "INFO\t", log.Ldate|log.Ltime)
	ErrorLogger = log.New(logger, "ERROR\t", log.Lshortfile|log.Ldate|log.Ltime)
	log.SetOutput(logger)
	return nil
}
