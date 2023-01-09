package main

import (
	"fmt"
	"io/fs"
	"os"
	"testing"
)

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

	testFile := gameTestRoot + string(os.PathSeparator) + "file1"
	payload := "Hello World"
	err = rootFs.WriteFile(testFile, []byte(payload), os.ModePerm)
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
			break
		} else {
			fmt.Println(msg.Message)
		}
	}

	payloadPath := rootDir + SAVE_FOLDER + string(os.PathSeparator) + t.Name() + string(os.PathSeparator) + "saves" + string(os.PathSeparator) + "file1"
	fmt.Println("Looking at : " + payloadPath)
	result, err := fs.ReadFile(rootFs, payloadPath)
	if err != nil {
		t.Error(err)
	}

	if string(result) != payload {
		t.Error("payload failure...")
	}
}

func injectTestGameDef(dm GameDefManager, testDataRoot string, t *testing.T) {
	genericDatapath := []*Datapath{
		{
			Path:    testDataRoot,
			Exts:    []string{},
			Ignore:  []string{},
			Parent:  "saves",
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
