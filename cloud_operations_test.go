package main

import (
	"fmt"
	"os"
	"testing"
)

func injectTestGameDef(dm GameDefManager, testDataRoot string, t *testing.T) {
	genericDatapath := []*Datapath{
		{
			Path:    testDataRoot,
			Exts:    []string{},
			Ignore:  []string{},
			Parent:  t.Name(),
			NetAuth: CloudOperationAll,
		},
	}

	dm.GetGameDefMap()[t.Name()] = &GameDef{
		DisplayName:          t.Name(),
		SteamId:              "0",
		SavesCrossCompatible: true,
		WinPath:              genericDatapath,
		DarwinPath:           genericDatapath,
		LinuxPath:            genericDatapath,
	}
}

func TestBasic(t *testing.T) {
	service = &LocalMemFsCloudDriver{}
	service.InitDriver()
	rootFs := service.(*LocalMemFsCloudDriver).GetRootFs()
	rootDir := service.(*LocalMemFsCloudDriver).GetRoot()

	fmt.Println("Setting up " + rootDir)

	testDataRoot := rootDir + SAVE_FOLDER + string(os.PathSeparator) + "testData"

	err := rootFs.MkdirAll(testDataRoot, 0755)
	if err != nil {
		t.Error(err)
	}

	gameTestRoot := testDataRoot + string(os.PathSeparator) + t.Name()
	fmt.Println("Making Dir ... " + gameTestRoot)
	err = rootFs.MkdirAll(gameTestRoot, 0755)
	if err != nil {
		t.Error(err)
	}

	dm := &MockGameDefManager{
		gamedefs: make(map[string]*GameDef),
		rootFs:   rootFs,
	}
	injectTestGameDef(dm, gameTestRoot, t)

	err = rootFs.WriteFile(gameTestRoot+string(os.PathSeparator)+"file1", []byte("Hello World"), os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	ops := &Options{
		Gamenames: []string{t.Name()},
		NoGUI:     []bool{true},
	}

	channels := &ChannelProvider{
		logs:   make(chan Message, 100),
		cancel: make(chan Cancellation, 1),
	}

	mockFs := &MockLocalFs{
		rootFs: rootFs,
	}

	go CliMain(ops, dm, channels, mockFs)

	for {
		msg := <-channels.logs
		if msg.Err != nil {
			t.Error(msg.Err)
		} else if msg.Finished {
			return
		} else {
			fmt.Println(msg.Message)
		}
	}
}
