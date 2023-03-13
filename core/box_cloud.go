package core

import (
	"context"
	"os/exec"
)

type BoxStorage struct {
}

func (gs *BoxStorage) GetName() string {
	return "opencloudsave-box"
}

func (gs *BoxStorage) GetCreationCommand(ctx context.Context) *exec.Cmd {
	return makeCommand(ctx, getCloudApp(), "config", "create", gs.GetName(), "box")
}

var boxStorage *BoxStorage

func GetBoxStorage() Storage {
	if boxStorage == nil {
		boxStorage = &BoxStorage{}
	}

	return boxStorage
}
