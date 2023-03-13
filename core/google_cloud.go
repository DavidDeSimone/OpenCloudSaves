package core

import (
	"context"
	"os/exec"
)

type GoogleStorage struct {
}

func (gs *GoogleStorage) GetName() string {
	return "opencloudsave-googledrive"
}

func (gs *GoogleStorage) GetCreationCommand(ctx context.Context) *exec.Cmd {
	return makeCommand(ctx, getCloudApp(), "config", "create", gs.GetName(), "drive", "scope=drive.file")
}

var gdrive *GoogleStorage

func GetGoogleDriveStorage() Storage {
	if gdrive == nil {
		gdrive = &GoogleStorage{}
	}

	return gdrive
}
