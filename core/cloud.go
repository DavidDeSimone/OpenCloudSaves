package core

import (
	"context"
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
const printCommands = true

// This is a real hack, but we fallback to $PATH if we can't
// find rclone locally in linux. This is really only for the
// flatpak - we control what version of rclone will be on $PATH
// within the flatpak
var checkedLinuxPath = false
var relativeLinuxPath = true

type Storage interface {
	GetName() string
	GetCreationCommand(ctx context.Context) *exec.Cmd
}

type CloudManager struct {
}

type CloudOperationOptions struct {
	Verbose     bool
	DryRun      bool
	Include     string
	UpdateOnly  bool
	Checksum    bool
	CustomFlags string
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

func makeCommand(ctx context.Context, cmd_string string, arg ...string) *exec.Cmd {
	if printCommands {
		InfoLogger.Println("Running Command ", cmd_string, arg)
	}

	cmd := exec.CommandContext(ctx, cmd_string, arg...)
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

func (cm *CloudManager) CreateDriveIfNotExists(ctx context.Context, storage Storage) error {
	InfoLogger.Println("Checking if drive exists")
	if cm.ContainsStorageDrive(ctx, storage) {
		return nil
	}

	InfoLogger.Println("Creating new drive for storage")
	err := cm.MakeStorageDrive(ctx, storage)
	if err != nil {
		ErrorLogger.Println(err)
		return err
	}

	return nil
}

func (cm *CloudManager) ContainsStorageDrive(ctx context.Context, storage Storage) bool {
	cmd := makeCommand(ctx, getCloudApp(), "config", "dump")
	stdout, err := cmd.Output()

	if err != nil {
		InfoLogger.Println(err.Error())
		return false
	}
	InfoLogger.Println(string(stdout))

	var data map[string]interface{}
	err = json.Unmarshal(stdout, &data)
	if err != nil {
		ErrorLogger.Println(err)
		return false
	}

	_, ok := data[storage.GetName()]
	return ok
}

func (cm *CloudManager) MakeStorageDrive(ctx context.Context, storage Storage) error {
	cmd := storage.GetCreationCommand(ctx)
	InfoLogger.Println("Running creation command", cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}

	return nil
}

func (cm *CloudManager) DoesRemoteDirExist(ctx context.Context, storage Storage, remotePath string) (bool, error) {
	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	cmd := makeCommand(ctx, getCloudApp(), "lsjson", path+"/")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		ErrorLogger.Println(stderr.String())
		return false, nil
	}

	return true, nil
}

func (cm *CloudManager) MakeRemoteDir(ctx context.Context, storage Storage, remotePath string) error {
	path := fmt.Sprintf("%v:%v/", storage.GetName(), remotePath)
	cmd := makeCommand(ctx, getCloudApp(), "mkdir", path)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}

	return nil
}

func (cm *CloudManager) ListFiles(ctx context.Context, ops *CloudOperationOptions, localPath string) ([]CloudFile, error) {
	defaultFlag := "--use-json-log"
	include := defaultFlag
	if ops.Include != "" {
		include = fmt.Sprintf("--include=%v", ops.Include)
	}

	cmd := makeCommand(ctx, getCloudApp(), defaultFlag, include, "lsjson", localPath)
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

func (cm *CloudManager) DeleteCloudEntry(ctx context.Context, storage Storage) error {
	name := storage.GetName()
	cmd := makeCommand(ctx, getCloudApp(), "config", "delete", name)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}

	return nil
}

func (cm *CloudManager) ObscurePassword(ctx context.Context, password string) (string, error) {
	cmd := makeCommand(ctx, getCloudApp(), "obscure", password)
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

func (cm *CloudManager) PerformSyncOperation(ctx context.Context, storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	InfoLogger.Println("Performing Sync Operation")
	os.MkdirAll(localPath, os.ModePerm)
	exists, err := cm.DoesRemoteDirExist(ctx, storage, remotePath)
	if err != nil {
		return "", err
	}
	if !exists {
		err = cm.MakeRemoteDir(ctx, storage, remotePath)
		if err != nil {
			return "", err
		}
	}

	cloudperfs := GetCurrentCloudPerfsOrDefault()
	if cloudperfs.UseBiSync {
		return cm.bisyncDir(ctx, storage, ops, localPath, remotePath)
	} else {
		return cm.syncDir(ctx, storage, ops, localPath, remotePath)
	}
}

func (cm *CloudManager) syncDir(ctx context.Context, storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	exists, err := cm.DoesRemoteDirExist(ctx, storage, remotePath)
	if err != nil {
		return "", err
	}
	copy := ""
	if exists {
		exisitingUFlag := ops.UpdateOnly
		exisitingChecksumFlag := ops.Checksum

		ops.UpdateOnly = true
		ops.Checksum = true

		copy, err = cm.copy(ctx, storage, ops, path, localPath)
		if err != nil {
			return "", err
		}
		ops.UpdateOnly = exisitingUFlag
		ops.Checksum = exisitingChecksumFlag
	}
	result, err := cm.sync(ctx, storage, ops, localPath, path)
	if err != nil {
		return "", err
	}

	return copy + "\n" + result, nil
}

func (cm *CloudManager) copy(ctx context.Context, storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	return cm.syncAction(ctx, "copy", storage, ops, localPath, remotePath)
}

func (cm *CloudManager) sync(ctx context.Context, storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	return cm.syncAction(ctx, "sync", storage, ops, localPath, remotePath)
}

func (cm *CloudManager) syncAction(ctx context.Context, action string, storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	args := []string{"--use-json-log"}
	if ops.Verbose {
		args = append(args, "-vv")
	}

	if ops.DryRun {
		args = append(args, "--dry-run")
	}

	if ops.Include != "" {
		args = append(args, fmt.Sprintf("--include=%v", ops.Include))
	}

	if ops.UpdateOnly {
		args = append(args, "-u")
	}

	if ops.Checksum {
		args = append(args, "--checksum")
	}

	if len(ops.CustomFlags) > 0 {
		trimmed := strings.TrimSpace(ops.CustomFlags)
		flags := strings.Split(trimmed, " ")
		args = append(args, flags...)
	}

	args = append(args, action, localPath, remotePath)

	cmd := makeCommand(ctx, getCloudApp(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var stdout strings.Builder
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf(stderr.String())
	}

	result := stderr.String()
	if result == "" {
		result = stdout.String()
	}

	// Rclone reports the information we want to display to the user
	// via stderr instead of stdout. To capture all of stderr, we use
	// the pipe instead of CombinedOutput, so we will return stderr
	return result, nil
}

func (cm *CloudManager) bisyncDir(ctx context.Context, storage Storage, ops *CloudOperationOptions, localPath string, remotePath string) (string, error) {
	args := []string{"--use-json-log"}
	if ops.Verbose {
		args = append(args, "-vv")
	}

	if ops.DryRun {
		args = append(args, "--dry-run")
	}

	if ops.Include != "" {
		args = append(args, fmt.Sprintf("--include=%v", ops.Include))
	}

	if len(ops.CustomFlags) > 0 {
		trimmed := strings.TrimSpace(ops.CustomFlags)
		flags := strings.Split(trimmed, " ")
		args = append(args, flags...)
	}

	path := fmt.Sprintf("%v:%v", storage.GetName(), remotePath)
	args = append(args, "bisync", localPath, path)

	cmd := makeCommand(ctx, getCloudApp(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var stdout strings.Builder
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		exiterr := err.(*exec.ExitError)
		if exiterr.ExitCode() == 2 {
			InfoLogger.Println("Need to run resync")
			args = append(args, "--resync")
			cmd := makeCommand(ctx, getCloudApp(), args...)
			var resyncstderr strings.Builder
			cmd.Stderr = &resyncstderr

			var resyncstdout strings.Builder
			cmd.Stdout = &resyncstdout

			err = cmd.Run()
			if err != nil {
				return "", fmt.Errorf(resyncstderr.String())
			}

			result := resyncstderr.String()
			if result == "" {
				result = resyncstdout.String()
			}
			return result, nil
		}

		return "", fmt.Errorf(stderr.String())
	}

	result := stderr.String()
	if result == "" {
		result = stdout.String()
	}

	// Rclone reports the information we want to display to the user
	// via stderr instead of stdout. To capture all of stderr, we use
	// the pipe instead of CombinedOutput, so we will return stderr
	return result, nil
}
