package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const GOOGLE = 0
const ONEDRIVE = 1
const DROPBOX = 2
const BOX = 3
const NEXT = 4

type CloudPerfs struct {
	Cloud         int  `json:"cloud"`
	PerformDryRun bool `json:performDryRun`
}

func getCloudPerfDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	fmt.Println("Looking - " + configDir)
	return configDir + string(os.PathSeparator) + APP_NAME + string(os.PathSeparator), nil

}

func getCloudPath() (string, error) {
	dir, err := getCloudPerfDir()
	if err != nil {
		return "", err
	}

	return dir + "opencloud_perfs.json", nil
}

func readCloudPerfs() (*CloudPerfs, error) {
	cloudperfs := &CloudPerfs{}
	path, err := getCloudPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, cloudperfs)
	if err != nil {
		return nil, err
	}
	return cloudperfs, nil
}

func writeCloudPerfs(cloudperfs *CloudPerfs) error {
	data, err := json.Marshal(cloudperfs)
	if err != nil {
		return err
	}
	path, err := getCloudPath()
	if err != nil {
		return err
	}

	dir, err := getCloudPerfDir()
	if err != nil {
		return err
	}

	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, os.ModePerm)
}

func GetCurrentCloudPerfs() (*CloudPerfs, error) {
	cloudperfs, err := readCloudPerfs()
	if err != nil {
		return nil, err
	}

	return cloudperfs, nil
}

func GetCurrentCloudPerfsOrDefault() *CloudPerfs {
	cloudperfs, err := GetCurrentCloudPerfs()
	if err != nil {
		cloudperfs = &CloudPerfs{
			Cloud:         GOOGLE,
			PerformDryRun: true,
		}
	}

	return cloudperfs
}

func GetCurrentCloudStorage() (Storage, error) {
	cloudperfs, err := GetCurrentCloudPerfs()
	if err != nil {
		return nil, err
	}

	switch cloudperfs.Cloud {
	case GOOGLE:
		return &GoogleStorage{}, nil
	case ONEDRIVE:
		return &OneDriveStorage{}, nil
	case DROPBOX:
		return &DropBoxStorage{}, nil
	case BOX:
		return &BoxStorage{}, nil
	case NEXT:
		return &NextCloudStorage{}, nil
	default:
		return nil, fmt.Errorf("failed to identify cloud solution")
	}
}

func UpdateCloudProvider(cloud int) error {
	cloudperfs, err := GetCurrentCloudPerfs()
	if err != nil {
		return err
	}
	cloudperfs.Cloud = cloud
	return CommitCloudPerfs(cloudperfs)
}

func CommitCloudPerfs(cloudperfs *CloudPerfs) error {
	return writeCloudPerfs(cloudperfs)
}
