package core

import "os/exec"

type NextCloudStorage struct {
	Url          string `json:"url"`
	User         string `json:"user"`
	Pass         string `json:"pass"`
	Bearer_token string `json:"bearer_token"`
}

func (gs *NextCloudStorage) GetName() string {
	return "opencloudsave-nextcloud"
}

func (gs *NextCloudStorage) GetCreationCommand() *exec.Cmd {
	args := []string{"config", "create", gs.GetName(), "webdav", "vendor=nextcloud"}
	if gs.Url != "" {
		args = append(args, "url="+gs.Url)
	}
	if gs.User != "" {
		args = append(args, "user="+gs.User)
	}
	if gs.Pass != "" {
		args = append(args, "pass="+gs.Pass)
	}
	if gs.Bearer_token != "" {
		args = append(args, "bearer_token="+gs.Bearer_token)
	}

	return makeCommand(getCloudApp(), args...)
}

var nextCloudStorage *NextCloudStorage

func DeleteNextCloudStorage() error {
	storage := &NextCloudStorage{}
	cm := MakeCloudManager()
	return cm.DeleteCloudEntry(storage)
}

func SetNextCloudStorage(ns *NextCloudStorage) {
	nextCloudStorage = ns
}

func GetNextCloudStorage() Storage {
	if nextCloudStorage == nil {
		nextCloudStorage = &NextCloudStorage{}
	}

	return nextCloudStorage
}
