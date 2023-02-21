package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const ToplevelCloudFolder = "opencloudsaves/"

// Used for debugging
const printCommands = false

// This is a real hack, but we fallback to $PATH if we can't
// find rclone locally in linux. This is really only for the
// flatpak - we control what version of rclone will be on $PATH
// within the flatpak
var checkedLinuxPath = false
var relativeLinuxPath = true

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

type CloudFile struct {
	Path     string
	Name     string
	Size     int64
	MimeType string
	ModTime  string
	IsDir    bool
}

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
	if printCommands {
		fmt.Println("Running Command ", cmd_string, arg)
	}

	cmd := exec.Command(cmd_string, arg...)
	StripWindow(cmd)
	return cmd
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
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}

	return nil
}

func (cm *CloudManager) DoesRemoteDirExist(storage Storage, remotePath string) (bool, error) {
	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	cmd := makeCommand(getCloudApp(), "lsjson", path+"/")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
		return false, nil
	}

	return true, nil
}

func (cm *CloudManager) MakeRemoteDir(storage Storage, remotePath string) error {
	path := fmt.Sprintf("%v:%v/", storage.GetName(), remotePath)
	cmd := makeCommand(getCloudApp(), "mkdir", path)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}

	return nil
}

func (cm *CloudManager) ListFiles(ops *CloudOperationOptions, localPath string) ([]CloudFile, error) {
	defaultFlag := "--use-json-log"
	include := defaultFlag
	if ops.Include != "" {
		include = fmt.Sprintf("--include=%v", ops.Include)
	}

	fmt.Println("Running Command ", getCloudApp(), defaultFlag, include, "lsjson", localPath)
	cmd := makeCommand(getCloudApp(), defaultFlag, include, "lsjson", localPath)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var stdout strings.Builder
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf(stderr.String())
	}

	arr := []CloudFile{}
	err = json.Unmarshal([]byte(stdout.String()), &arr)
	if err != nil {
		return nil, err
	}

	return arr, nil
}

func (cm *CloudManager) DeleteCloudEntry(storage Storage) error {
	name := storage.GetName()
	cmd := makeCommand(getCloudApp(), "config", "delete", name)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}

	return nil
}

func (cm *CloudManager) ObscurePassword(password string) (string, error) {
	cmd := makeCommand(getCloudApp(), "obscure", password)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var output strings.Builder
	cmd.Stdout = &output

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf(stderr.String())
	}
	return output.String(), nil
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
	args := []string{"--use-json-log"}
	if ops.Verbose {
		args = append(args, "-v")
	}

	if ops.DryRun {
		args = append(args, "--dry-run")
	}

	if ops.Include != "" {
		args = append(args, fmt.Sprintf("--include=%v", ops.Include))
	}

	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	args = append(args, "sync", localPath, path)

	cmd := makeCommand(getCloudApp(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var stdout strings.Builder
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf(stderr.String())
	}

	// Rclone reports the information we want to display to the user
	// via stderr instead of stdout. To capture all of stderr, we use
	// the pipe instead of CombinedOutput, so we will return stderr
	return stderr.String(), nil
}

func (cm *CloudManager) bisyncDir(storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	args := []string{"--use-json-log"}
	if ops.Verbose {
		args = append(args, "--verbose")
	}

	if ops.DryRun {
		args = append(args, "--dry-run")
	}

	if ops.Include != "" {
		args = append(args, fmt.Sprintf("--include=%v", ops.Include))
	}

	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	args = append(args, "bisync", localPath, path)

	cmd := makeCommand(getCloudApp(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var stdout strings.Builder
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		exiterr := err.(*exec.ExitError)
		if exiterr.ExitCode() == 2 {
			fmt.Println("Need to run resync")
			args = append(args, "--resync")
			cmd := makeCommand(getCloudApp(), args...)
			var resyncstderr strings.Builder
			cmd.Stderr = &resyncstderr

			var resyncstdout strings.Builder
			cmd.Stdout = &resyncstdout

			err = cmd.Run()
			if err != nil {
				return "", fmt.Errorf(resyncstderr.String())
			}
			return resyncstderr.String(), nil
		}

		return "", fmt.Errorf(stderr.String())
	}

	// Rclone reports the information we want to display to the user
	// via stderr instead of stdout. To capture all of stderr, we use
	// the pipe instead of CombinedOutput, so we will return stderr
	return stderr.String(), nil
}
