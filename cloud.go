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

// This is a real hack, but we fallback to $PATH if we can't
// find rclone locally in linux. This is really only for the
// flatpak - we control what version of rclone will be on $PATH
// within the flatpak
var checkedLinuxPath = false
var relativeLinuxPath = true

func getCloudApp() string {
	switch runtime.GOOS {
	case "linux":
		if !checkedLinuxPath {
			_, err := os.Stat("./bin/rclone")
			if err != nil {
				relativeLinuxPath = false
			}
			checkedLinuxPath = true
		}

		if relativeLinuxPath {
			return "./bin/rclone"
		} else {
			return "rclone"
		}

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
	Include string
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
	fmt.Println("Checking if drive exists...")
	if cm.ContainsStorageDrive(storage) {
		fmt.Println("Not creating drive...")
		return nil
	}

	fmt.Println("Creating new drive for storage....")
	err := cm.MakeStorageDrive(storage)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
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
	fmt.Println("Getting creation command...")
	cmd := storage.GetCreationCommand()
	fmt.Println("Running creation command", cmd)
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

type CloudFile struct {
	Path     string
	Name     string
	Size     int64
	MimeType string
	ModTime  string
	IsDir    bool
}

func (cm *CloudManager) ListFiles(ops *CloudOperationOptions, localPath string) ([]CloudFile, error) {
	defaultFlag := "--use-json-log"
	include := defaultFlag
	if ops.Include != "" {
		include = fmt.Sprintf("--include=%v", ops.Include)
	}

	fmt.Println("Running Command ", getCloudApp(), defaultFlag, include, "lsjson", localPath)
	cmd := makeCommand(getCloudApp(), defaultFlag, include, "lsjson", localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	arr := []CloudFile{}
	err = json.Unmarshal(output, &arr)
	if err != nil {
		return nil, err
	}

	return arr, nil
}

func (cm *CloudManager) PerformSyncOperation(storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	fmt.Println("Performing Sync Operation....")
	os.MkdirAll(localPath, os.ModePerm)
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

	cloudperfs := GetCurrentCloudPerfsOrDefault()
	if cloudperfs.UseBiSync {
		return cm.bisyncDir(storage, ops, localPath, remotePath)
	} else {
		return cm.syncDir(storage, ops, localPath, remotePath)
	}
}

// @TODO support cancellation
func (cm *CloudManager) syncDir(storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	// We can't pass an empty string as a flag to the rclone command, but we
	// can pass the same flag multiple times. We use this as a hack to enable
	// conditional commands with a varargs function. There is likely a better way to
	// do this, but this should be generally low cost.
	defaultFlag := "--use-json-log"

	verboseString := defaultFlag
	if ops.Verbose {
		verboseString = "-v"
	}

	dryRunString := defaultFlag
	if ops.DryRun {
		dryRunString = "--dry-run"
	}

	include := defaultFlag
	if ops.Include != "" {
		include = fmt.Sprintf("--include=%v", ops.Include)
	}

	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	fmt.Println("Running Command ", getCloudApp(), defaultFlag, verboseString, dryRunString, include, "sync", localPath, path)
	cmd := makeCommand(getCloudApp(), defaultFlag, verboseString, dryRunString, include, "sync", localPath, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func (cm *CloudManager) bisyncDir(storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {

	// We can't pass an empty string as a flag to the rclone command, but we
	// can pass the same flag multiple times. We use this as a hack to enable
	// conditional commands with a varargs function. There is likely a better way to
	// do this, but this should be generally low cost.
	defaultFlag := "--use-json-log"

	verboseString := defaultFlag
	if ops.Verbose {
		verboseString = "-v"
	}

	dryRunString := defaultFlag
	if ops.DryRun {
		dryRunString = "--dry-run"
	}

	include := defaultFlag
	if ops.Include != "" {
		include = fmt.Sprintf("--include=%v", ops.Include)
	}

	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	fmt.Println("Running Command ", getCloudApp(), defaultFlag, verboseString, dryRunString, include, "bisync", localPath, path)
	cmd := makeCommand(getCloudApp(), defaultFlag, verboseString, dryRunString, include, "bisync", localPath, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		exiterr := err.(*exec.ExitError)
		if exiterr.ExitCode() == 2 {
			// @TODO - in this case, I want to explain sync vs bisync to the user and let them choose
			fmt.Println("Need to run resync")
			fmt.Println("Running Command ", getCloudApp(), defaultFlag, verboseString, dryRunString, include, "--resync", "bisync", localPath, path)
			cmd := makeCommand(getCloudApp(), defaultFlag, verboseString, dryRunString, include, "bisync", "--resync", localPath, path)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return "", err
			}
			return string(output), nil
		}

		return "", err
	}

	return string(output), nil
}
