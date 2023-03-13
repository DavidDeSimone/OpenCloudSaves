package core

import (
	"context"
	"os/exec"
)

type DropBoxStorage struct {
}

func (gs *DropBoxStorage) GetName() string {
	return "opencloudsave-dropbox"
}

func (gs *DropBoxStorage) GetCreationCommand(ctx context.Context) *exec.Cmd {
	return makeCommand(ctx, getCloudApp(), "config", "create", gs.GetName(), "dropbox")
}

var dropboxStorage *DropBoxStorage

func GetDropBoxStorage() Storage {
	if dropboxStorage == nil {
		dropboxStorage = &DropBoxStorage{}
	}

	return dropboxStorage
}
