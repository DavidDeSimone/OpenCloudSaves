package main

import (
	"bytes"
	"fmt"
	"io/fs"
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

func syncRun(t *testing.T, srv CloudDriver) {
	// Override global cloud service
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

	go CliMain(srv, ops, dm, channels, SyncOp)

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

	os.RemoveAll(os.TempDir() + "/" + t.Name())
	os.RemoveAll(os.TempDir() + "/" + t.Name() + ".zip")
}

func validate(src string, dest string, t *testing.T) {
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		destPath := dest + "/"
		origContent, err := os.ReadFile(destPath + d.Name())
		if err != nil {
			return err
		}
		cloudContent, err := os.ReadFile(src + "/" + d.Name())
		if err != nil {
			return err
		}

		if bytes.Compare(origContent, cloudContent) != 0 {
			t.Error("cloud file " + d.Name() + "not in sync")
		}

		return nil
	})

	if err != nil {
		t.Error(err)
	}
}

func validateLocalAndRemoteInSync(service *MockCloudDriver, t *testing.T) {
	res, err := fs.ReadFile(service.fs, "root/steamsave/"+t.Name()+"/"+t.Name()+".zip")
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(os.TempDir()+"/"+t.Name()+".zip", res, os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	unzipSource(os.TempDir()+"/"+t.Name()+".zip", os.TempDir()+"/"+t.Name())
	validate(os.TempDir()+"/"+t.Name(), "tests/"+t.Name()+"/", t)
	validate("tests/"+t.Name()+"/", os.TempDir()+"/"+t.Name(), t)

}

func TestBasic(t *testing.T) {
	service := &MockCloudDriver{}
	service.InitDriver()

	setupTest(t)
	syncRun(t, service)
	validateLocalAndRemoteInSync(service, t)
}

func TestLocalFileChanged(t *testing.T) {
	service := &MockCloudDriver{}
	service.InitDriver()

	setupTest(t)
	syncRun(t, service)
	err := os.WriteFile("tests/"+t.Name()+"/save1", []byte("New Content"), os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	syncRun(t, service)
	validateLocalAndRemoteInSync(service, t)
}

func TestLocalFileAdded(t *testing.T) {
	service := &MockCloudDriver{}
	service.InitDriver()

	setupTest(t)
	syncRun(t, service)
	err := os.WriteFile("tests/"+t.Name()+"/newFile", []byte("New Content"), os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	syncRun(t, service)
	validateLocalAndRemoteInSync(service, t)
}

func TestRemoteFileChanged(t *testing.T) {
	service := &MockCloudDriver{}
	service.InitDriver()

	setupTest(t)
	syncRun(t, service)

	newDir := os.TempDir() + "/" + t.Name() + "/988s9a/"
	err := os.MkdirAll(newDir, os.ModePerm)
	if err != nil {
		t.Error(err)
	}
	err = copy("tests/"+t.Name(), newDir)
	if err != nil {
		t.Error(err)
	}

	err = os.WriteFile(newDir+"save1", []byte("New Content for new world"), os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	_, err = zipSource(newDir, newDir+"source.zip", func(s string) bool { return false })
	if err != nil {
		t.Error(err)
	}
	content, err := os.ReadFile(newDir + "source.zip")
	if err != nil {
		t.Error(err)
	}
	fmt.Println("root/steamsave/" + t.Name() + "/" + t.Name() + "/" + t.Name() + ".zip")
	err = service.fs.WriteFile("root/steamsave/"+t.Name()+"/"+t.Name()+".zip", content, os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	syncRun(t, service)
	validateLocalAndRemoteInSync(service, t)
}

func TestLocalFileRemoved(t *testing.T) {
	service := &MockCloudDriver{}
	service.InitDriver()

	setupTest(t)
	syncRun(t, service)
	err := os.Remove("tests/" + t.Name() + "/save1")
	if err != nil {
		t.Error(err)
	}

	syncRun(t, service)
	validateLocalAndRemoteInSync(service, t)
}
