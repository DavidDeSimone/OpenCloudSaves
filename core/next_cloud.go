package core

import "os/exec"

type NextCloudStorage struct {
}

func (gs *NextCloudStorage) GetName() string {
	return "opencloudsave-nextcloud"
}

func (gs *NextCloudStorage) GetCreationCommand() *exec.Cmd {
	return makeCommand(getCloudApp(), "config", "create", gs.GetName(), "webdav", "vendor=nextcloud")
}

var nextCloudStorage *NextCloudStorage

func GetNextCloudStorage() Storage {
	if nextCloudStorage == nil {
		nextCloudStorage = &NextCloudStorage{}
	}

	return nextCloudStorage
}
