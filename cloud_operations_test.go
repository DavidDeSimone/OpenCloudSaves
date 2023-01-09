package main

import (
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/DavidDeSimone/memfs"
)

func run(t *testing.T, testEnv *TestEnv) {

	go CliMain(testEnv.ops, testEnv.dm, testEnv.channels, testEnv.mockFs)

	for {
		msg := <-testEnv.channels.logs
		if msg.Err != nil {
			t.Error(msg.Err)
		} else if msg.Finished {
			break
		} else {
			fmt.Println(msg.Message)
		}
	}
}

type TestEnv struct {
	rootFs   *memfs.FS
	rootDir  string
	dm       *MockGameDefManager
	ops      *Options
	channels *ChannelProvider
	mockFs   *MockLocalFs
}

func bootstrapTestingEnv(t *testing.T) *TestEnv {
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

	return &TestEnv{
		rootFs:   rootFs,
		rootDir:  rootDir,
		ops:      ops,
		dm:       dm,
		channels: channels,
		mockFs:   mockFs,
	}
}

func TestBasic(t *testing.T) {
	testEnv := bootstrapTestingEnv(t)

	testDataRoot := testEnv.rootDir + SAVE_FOLDER + string(os.PathSeparator) + "testData"

	err := testEnv.rootFs.MkdirAll(testDataRoot, 0755)
	if err != nil {
		t.Error(err)
	}

	gameTestRoot := testDataRoot + string(os.PathSeparator) + t.Name()
	testFile := gameTestRoot + string(os.PathSeparator) + "file1"
	payload := "Hello World"
	err = testEnv.rootFs.WriteFile(testFile, []byte(payload), os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	rootName := t.Name()
	t.Run("Mock", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			run(t, testEnv)

			payloadPath := testEnv.rootDir + SAVE_FOLDER + string(os.PathSeparator) + rootName + string(os.PathSeparator) + "saves" + string(os.PathSeparator) + "file1"
			fmt.Println("Looking at : " + payloadPath)
			result, err := fs.ReadFile(testEnv.rootFs, payloadPath)
			if err != nil {
				t.Error(err)
			}

			if string(result) != payload {
				t.Error("payload failure...")
			}
		}
	})

	t.Run("Local Mutate Triggers Upload", func(t *testing.T) {
		info, err := fs.Stat(testEnv.rootFs, testFile)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(info.ModTime())

		newPayload := "New Payload"
		err = testEnv.rootFs.WriteFile(testFile, []byte(newPayload), os.ModePerm)
		if err != nil {
			t.Error(err)
		}

		info, err = fs.Stat(testEnv.rootFs, testFile)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(info.ModTime())

		run(t, testEnv)

		payloadPath := testEnv.rootDir + SAVE_FOLDER + string(os.PathSeparator) + rootName + string(os.PathSeparator) + "saves" + string(os.PathSeparator) + "file1"
		data, err := fs.ReadFile(testEnv.rootFs, payloadPath)
		if err != nil {
			t.Error(err)
		}

		if string(data) != newPayload {
			t.Errorf("Upload Failed with data -> (%v, %v)\n", string(data), newPayload)
		}
	})
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
