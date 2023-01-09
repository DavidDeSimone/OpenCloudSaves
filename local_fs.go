package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"os"
)

type LocalFs interface {
	WriteFile(path string, data []byte, mode fs.FileMode) error
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
	GetFileHash(path string) (string, error)
}

type DefaultLocalFs struct {
}

var defaultFs *DefaultLocalFs

func GetDefaultLocalFs() *DefaultLocalFs {
	if defaultFs == nil {
		defaultFs = &DefaultLocalFs{}
	}

	return defaultFs
}

func (d *DefaultLocalFs) WriteFile(path string, data []byte, mode fs.FileMode) error {
	return os.WriteFile(path, data, mode)
}

func (d *DefaultLocalFs) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (d *DefaultLocalFs) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (d *DefaultLocalFs) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (d *DefaultLocalFs) GetFileHash(path string) (string, error) {
	return getFileHash(path)
}

func getFileHash(fileName string) (string, error) {
	f, err := os.Open(fileName)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
