package main

import "os/exec"

type GoogleStorage struct {
}

func (gs *GoogleStorage) GetName() string {
	return "opencloudsave-googledrive"
}

func (gs *GoogleStorage) GetCreationCommand() *exec.Cmd {
	return makeCommand(getCloudApp(), "config", "create", gs.GetName(), "scope=drive.file")
}

var gdrive *GoogleStorage

func GetGoogleDriveStorage() Storage {
	if gdrive == nil {
		gdrive = &GoogleStorage{}
	}

	return gdrive
}
