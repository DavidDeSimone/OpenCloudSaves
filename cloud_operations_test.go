package main

import (
	"fmt"
	"io/fs"
	"os"
	"testing"
)

type MockGameDefManager struct {
	gamedefs map[string]*GameDef
	rootFs   fs.FS
}

func (m *MockGameDefManager) ApplyUserOverrides() error {
	return nil
}
func (m *MockGameDefManager) CommitUserOverrides() error {
	return nil
}
func (m *MockGameDefManager) AddUserOverride(key string, jsonOverride string) error {
	return nil
}
func (m *MockGameDefManager) GetGameDefMap() map[string]*GameDef {
	return m.gamedefs
}

// @TODO
func (m *MockGameDefManager) GetFilesForGame(id string, parent string) (map[string]SyncFile, error) {
	return make(map[string]SyncFile), nil
}

// @TODO
func (m *MockGameDefManager) GetSyncpathForGame(id string) ([]Datapath, error) {
	return nil, nil
}

func injectTestGameDef(dm GameDefManager, testDataRoot string, t *testing.T) {
	dm.GetGameDefMap()[t.Name()] = &GameDef{
		DisplayName:          t.Name(),
		SteamId:              "0",
		SavesCrossCompatible: true,
		WinPath: []*Datapath{
			{
				Path:    testDataRoot,
				Exts:    []string{},
				Ignore:  []string{},
				Parent:  t.Name(),
				NetAuth: CloudOperationAll,
			},
		},
		DarwinPath: []*Datapath{
			{
				Path:    testDataRoot,
				Exts:    []string{},
				Ignore:  []string{},
				Parent:  t.Name(),
				NetAuth: CloudOperationAll,
			},
		},
		LinuxPath: []*Datapath{
			{
				Path:    testDataRoot,
				Exts:    []string{},
				Ignore:  []string{},
				Parent:  t.Name(),
				NetAuth: CloudOperationAll,
			},
		},
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

	dm := &MockGameDefManager{}
	injectTestGameDef(dm, gameTestRoot, t)

	err = rootFs.WriteFile(gameTestRoot+string(os.PathSeparator)+"file1", []byte("Hello World"), os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	ops := &Options{
		Gamenames: []string{t.Name()},
		NoGUI:     []bool{true},
	}

	logs := make(chan Message, 100)
	cancel := make(chan Cancellation, 1)
	go CliMain(ops, dm, logs, cancel)

	for {
		msg := <-logs
		if msg.Err != nil {
			t.Error(msg.Err)
		} else if msg.Finished {
			return
		} else {
			fmt.Println(msg.Message)
		}
	}
}
