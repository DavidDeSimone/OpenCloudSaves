package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
		return GetMacOsPath()
	default:
		log.Fatal("Unsupported Platform")
	}

	return "Unsupported Platform"
}

func makeCommand(cmd_string string, arg ...string) *exec.Cmd {
	cmd := exec.Command(cmd_string, arg...)
	StripWindow(cmd)
	return cmd
}

type Storage interface {
	GetName() string
	GetCreationCommand() *exec.Cmd
}

type CloudManager struct {
}

type CloudOperationOptions struct {
	Verbose bool
	DryRun  bool
}

func GetDefaultCloudOptions() *CloudOperationOptions {
	return &CloudOperationOptions{
		Verbose: false,
		DryRun:  false,
	}
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
	cmd := makeCommand(getCloudApp(), "config", "dump")
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
	cmd := makeCommand(getCloudApp(), "lsjon", path+"/")
	err := cmd.Run()
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (cm *CloudManager) MakeRemoteDir(storage Storage, remotePath string) error {
	path := fmt.Sprintf("%v:%v/", storage.GetName(), remotePath)
	cmd := makeCommand(getCloudApp(), "mkdir", path)
	return cmd.Run()
}

func (cm *CloudManager) BisyncDir(storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	fmt.Println("Performing BiSync....")
	_, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}

	exists, err := cm.DoesRemoteDirExist(storage, remotePath)
	if err != nil {
		return "", err
	}
	if !exists {
		err = cm.MakeRemoteDir(storage, remotePath)
		if err != nil {
			return "", err
		}
	}

	defaultFlag := "--use-json-log"

	verboseString := defaultFlag
	if ops.Verbose {
		verboseString = "-v"
	}

	dryRunString := defaultFlag
	if ops.DryRun {
		dryRunString = "--dry-run"
	}

	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	fmt.Println("Running Command ", getCloudApp(), defaultFlag, verboseString, dryRunString, "bisync", localPath, path)
	cmd := makeCommand(getCloudApp(), defaultFlag, verboseString, dryRunString, "bisync", localPath, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		exiterr := err.(*exec.ExitError)
		if exiterr.ExitCode() == 2 {
			fmt.Println("Need to run resync")
			cmd := makeCommand(getCloudApp(), defaultFlag, verboseString, dryRunString, "bisync", "--resync", localPath, path)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return "", err
			}
			fmt.Println("Output: " + string(output))
			return string(output), nil
		}

		return "", err
	}

	return string(output), nil
}
