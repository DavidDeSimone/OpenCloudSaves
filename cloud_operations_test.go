package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func copy(source, destination string) error {
	var err error = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			var data, err1 = ioutil.ReadFile(filepath.Join(source, relPath))
			if err1 != nil {
				return err1
			}
			return ioutil.WriteFile(filepath.Join(destination, relPath), data, 0777)
		}
	})
	return err
}

func syncRun(t *testing.T) {
	// Override global cloud service
	service = &MockCloudDriver{}
	service.InitDriver()
	dm := MakeGameDefManager("tests/tests_useroverrides.json")
	ops := &Options{
		Gamenames: []string{t.Name()},
	}

	channels := &ChannelProvider{
		logs:     make(chan Message, 100),
		cancel:   make(chan Cancellation, 1),
		input:    make(chan SyncRequest, 10),
		output:   make(chan SyncResponse, 10),
		progress: make(chan ProgressEvent, 15),
	}

	go CliMain(ops, dm, channels, SyncOp)

	for {

		select {
		case result := <-channels.logs:
			if result.Finished {
				return
			}

			if result.Err != nil {
				fmt.Println(result.Err)
			} else {
				fmt.Println(result.Message)
			}
		case <-time.After(20 * time.Second):
			t.Error("timeout")
		}
	}
}

func setupTest(t *testing.T) {
	testPath := "tests/" + t.Name()
	os.RemoveAll(testPath)
	os.MkdirAll(testPath, os.ModePerm)
	copy("tests/data", testPath)
}

func TestBasic(t *testing.T) {
	setupTest(t)
	syncRun(t)
}
