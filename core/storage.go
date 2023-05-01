package core

import (
	"context"
	"os/exec"
)

type Storage interface {
	GetName() string
	GetCreationCommand(ctx context.Context) *exec.Cmd
}

func GetAllStorageProviders() []Storage {
	return []Storage{
		GetGoogleDriveStorage(),
		GetDropBoxStorage(),
		GetOneDriveStorage(),
		GetBoxStorage(),
		GetFtpDriveStorage(),
		GetNextCloudStorage(),
	}
}
