package core

import (
	"context"
	"os/exec"
)

type OneDriveStorage struct {
}

func (gs *OneDriveStorage) GetName() string {
	return "opencloudsave-onedrive"
}

func (gs *OneDriveStorage) GetCreationCommand(ctx context.Context) *exec.Cmd {
	return makeCommand(ctx, getCloudApp(), "config", "create", gs.GetName(), "onedrive", "drive_type=personal", "access_scopes=Files.ReadWrite,offline_access")
}

var onedrive *OneDriveStorage

func GetOneDriveStorage() Storage {
	if onedrive == nil {
		onedrive = &OneDriveStorage{}
	}

	return onedrive
}
