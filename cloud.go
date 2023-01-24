package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

var ToplevelCloudFolder = "opencloudsaves/"

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
	return exec.Command("./bin/rclone", "config", "create", gs.GetName(), "scope=drive.file")
}

var gdrive *GoogleStorage

func GetGoogleDriveStorage() Storage {
	if gdrive == nil {
		gdrive = &GoogleStorage{}
	}

	return gdrive
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
	cmd := exec.Command("./bin/rclone", "config", "dump")
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
	cmd := exec.Command("./bin/rclone", "lsjon", path+"/")
	err := cmd.Run()
	if err != nil {
		return false, nil
	}

	fmt.Println("Found Path " + path)
	return true, nil
}

func (cm *CloudManager) MakeRemoteDir(storage Storage, remotePath string) error {
	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	cmd := exec.Command("./bin/rclone", "mkdir", path)
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
		cmd := exec.Command("./bin/rclone", "bisync", "--resync", localPath, path)
		return cmd.Run()
	} else {
		path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
		cmd := exec.Command("./bin/rclone", "bisync", localPath, path)
		return cmd.Run()
	}
}
