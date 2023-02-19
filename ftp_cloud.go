package main

import "os/exec"

type FtpStorage struct {
	Host     string `json: host`
	UserName string `json: userName`
	Port     string `json: port`
	Password string `json: password`
}

func (ftpfs *FtpStorage) GetName() string {
	return "opencloudsave-ftp"
}

func (ftpfs *FtpStorage) GetCreationCommand() *exec.Cmd {
	return makeCommand(getCloudApp(), "config", "create", ftpfs.GetName(), "ftp", "host="+ftpfs.Host, "username="+ftpfs.UserName, "port="+ftpfs.Port, "password="+ftpfs.Password)
}

var ftpDrive *FtpStorage

func SetFtpDriveStorage(ftp *FtpStorage) {
	ftpDrive = ftp
}

func GetFtpDriveStorage() Storage {
	if ftpDrive == nil {
		ftpDrive = &FtpStorage{}
	}

	return ftpDrive
}
