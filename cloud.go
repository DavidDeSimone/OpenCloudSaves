package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

var ToplevelCloudFolder = "opencloudsaves/"

func getCloudApp() string {
	switch runtime.GOOS {
	case "linux":
		return "./bin/rclone"
	case "windows":
		return "./bin/rclone.exe"
	case "darwin":
		// @TODO in production macOS, we need to do this:
		// exe, _ := os.Executable()
		// dir := filepath.Dir(exe)
		// parent := filepath.Dir(dir)

		// return parent + "/Resources/rclone"
		return "./bin/rclone"
	default:
		log.Fatal("Unsupported Platform")
	}

	return "Unsupported Platform"
}

type Storage interface {
	GetName() string
	GetCreationCommand() *exec.Cmd
}

type GoogleStorage struct {
}

func (gs *GoogleStorage) GetName() string {
	return "opencloudsave-googledrive"
}

func (gs *GoogleStorage) GetCreationCommand() *exec.Cmd {
	return exec.Command(getCloudApp(), "config", "create", gs.GetName(), "scope=drive.file")
}

var gdrive *GoogleStorage

func GetGoogleDriveStorage() Storage {
	if gdrive == nil {
		gdrive = &GoogleStorage{}
	}

	return gdrive
}

type OneDriveStorage struct {
}

func (gs *OneDriveStorage) GetName() string {
	return "opencloudsave-onedrive"
}

func (gs *OneDriveStorage) GetCreationCommand() *exec.Cmd {
	return exec.Command(getCloudApp(), "config", "create", gs.GetName(), "onedrive")
}

var onedrive *OneDriveStorage

func GetOneDriveStorage() Storage {
	if onedrive == nil {
		onedrive = &OneDriveStorage{}
	}

	return onedrive
}

type CloudManager struct {
}

func MakeCloudManager() *CloudManager {
	return &CloudManager{}
}

func (cm *CloudManager) CreateDriveIfNotExists(storage Storage) error {
	if cm.ContainsStorageDrive(storage) {
		fmt.Println("Not creating drive...")
		return nil
	}

	fmt.Println("Creating new drive for storage....")
	return cm.MakeStorageDrive(storage)
}

func (cm *CloudManager) ContainsStorageDrive(storage Storage) bool {
	cmd := exec.Command(getCloudApp(), "config", "dump")
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	fmt.Println(string(stdout))

	var data map[string]interface{}
	err = json.Unmarshal(stdout, &data)
	if err != nil {
		fmt.Println(err)
		return false
	}

	_, ok := data[storage.GetName()]
	return ok
}

func (cm *CloudManager) MakeStorageDrive(storage Storage) error {
	cmd := storage.GetCreationCommand()
	return cmd.Run()
}

func (cm *CloudManager) DoesRemoteDirExist(storage Storage, remotePath string) (bool, error) {
	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	fmt.Println("Examining path " + path)
	cmd := exec.Command(getCloudApp(), "lsjon", path+"/")
	err := cmd.Run()
	if err != nil {
		return false, nil
	}

	fmt.Println("Found Path " + path)
	return true, nil
}

func (cm *CloudManager) MakeRemoteDir(storage Storage, remotePath string) error {
	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	cmd := exec.Command(getCloudApp(), "mkdir", path)
	return cmd.Run()
}

func (cm *CloudManager) BisyncDir(storage Storage, localPath string, remotePath string) error {
	exists, err := cm.DoesRemoteDirExist(storage, remotePath)
	if err != nil {
		return err
	}
	if !exists {
		err = cm.MakeRemoteDir(storage, remotePath)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
		fmt.Println("Bisyncing on " + path)
		cmd := exec.Command(getCloudApp(), "bisync", "--resync", localPath, path)
		return cmd.Run()
	} else {
		path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
		cmd := exec.Command(getCloudApp(), "bisync", localPath, path)
		return cmd.Run()
	}
}
