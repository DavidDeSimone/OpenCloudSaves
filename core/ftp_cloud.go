package core

import (
	"context"
	"os/exec"
)

type FtpStorage struct {
	Host     string `json:"host"`
	UserName string `json:"userName"`
	Port     string `json:"port"`
	Password string `json:"password"`
}

func (ftpfs *FtpStorage) GetName() string {
	return "opencloudsave-ftp"
}

func (ftpfs *FtpStorage) GetCreationCommand(ctx context.Context) *exec.Cmd {
	args := []string{"config", "create", ftpfs.GetName(), "ftp"}
	if ftpfs.Host != "" {
		args = append(args, "host="+ftpfs.Host)
	}
	if ftpfs.UserName != "" {
		args = append(args, "user="+ftpfs.UserName)
	}
	if ftpfs.Port != "" {
		args = append(args, "port="+ftpfs.Port)
	}
	if ftpfs.Password != "" {
		args = append(args, "pass="+ftpfs.Password)
	}

	return makeCommand(ctx, getCloudApp(), args...)
}

var ftpDrive *FtpStorage

func DeleteFtpDriveStorage(ctx context.Context) error {
	storage := &FtpStorage{}
	cm := MakeCloudManager()
	return cm.DeleteCloudEntry(ctx, storage)
}

func SetFtpDriveStorage(ftp *FtpStorage) {
	ftpDrive = ftp
}

func GetFtpDriveStorage() Storage {
	if ftpDrive == nil {
		ftpDrive = &FtpStorage{}
	}

	return ftpDrive
}
