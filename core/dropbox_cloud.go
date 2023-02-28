package core

import "os/exec"

type DropBoxStorage struct {
}

func (gs *DropBoxStorage) GetName() string {
	return "opencloudsave-dropbox"
}

func (gs *DropBoxStorage) GetCreationCommand() *exec.Cmd {
	return makeCommand(getCloudApp(), "config", "create", gs.GetName(), "dropbox")
}

var dropboxStorage *DropBoxStorage

func GetDropBoxStorage() Storage {
	if dropboxStorage == nil {
		dropboxStorage = &DropBoxStorage{}
	}

	return dropboxStorage
}
