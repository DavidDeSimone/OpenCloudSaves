package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"fyne.io/fyne"
	cont "fyne.io/fyne/container"
	"fyne.io/fyne/widget"
)

type GlobalOptions struct {
	UseLocalFsBackup     bool   `json:"use_local_fs_backup"`
	UseGoogleCloudBackup bool   `json:"use_google_cloud_backup"`
	LocalFsBackupPath    string `json:"local_fs_backup_path"`
}

const OptionsFileName string = "steamsaves_userconfig.json"

func GetDefaultLocalFsBackupDir() string {
	cachedir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}

	separator := string(os.PathSeparator)

	return cachedir + separator + APP_NAME + separator
}

func GetDefaultUserOptionsFileDir() string {
	userCache, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	separator := string(os.PathSeparator)

	return userCache + separator + APP_NAME + separator

}

func getDefaultUserOptionsFilePath() string {
	return GetDefaultUserOptionsFileDir() + OptionsFileName
}

var globalOptions *GlobalOptions = nil

func ReloadUserOptions() {
	content, err := os.ReadFile(getDefaultUserOptionsFilePath())
	if err != nil {
		globalOptions = &GlobalOptions{
			UseLocalFsBackup:     true,
			UseGoogleCloudBackup: true,
			LocalFsBackupPath:    GetDefaultLocalFsBackupDir(),
		}
	} else {
		err := json.Unmarshal(content, &globalOptions)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func GetGlobalOptions() *GlobalOptions {
	if globalOptions == nil {
		ReloadUserOptions()
	}

	return globalOptions
}

func CommitUserOptions() {
	data, err := json.Marshal(GetGlobalOptions())
	if err != nil {
		log.Fatal(err)
	}

	CreateDirIfNotExist(GetDefaultUserOptionsFileDir())
	err = os.WriteFile(getDefaultUserOptionsFilePath(), data, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}

func makeLocalFsBox(localOptions *GlobalOptions, optionHint string, defaultText string) fyne.CanvasObject {
	localFsCheckBox := widget.NewCheck(optionHint, func(b bool) {
		localOptions.UseLocalFsBackup = b
	})
	localFsCheckBox.Checked = localOptions.UseLocalFsBackup
	localFsPathBox := cont.NewHBox()

	localFsPath := widget.NewEntry()
	localFsPath.OnChanged = func(s string) {
		localOptions.LocalFsBackupPath = s
	}
	localFsPath.Text = localOptions.LocalFsBackupPath

	filePathLabel := widget.NewLabel("Destination File Path: ")

	localFsPathBox.Add(filePathLabel)
	localFsPathBox.Add(localFsPath)

	localFsBox := cont.NewVBox(localFsCheckBox, localFsPathBox)

	return localFsBox
}

func MakeOptionsScreen() fyne.CanvasObject {
	localOptions := GetGlobalOptions()

	container := cont.NewVBox()

	cloudDriverBox := cont.NewVBox()

	localFsBox := makeLocalFsBox(localOptions, "Enable Local Filesystem Backup", "")
	googleCheckBox := widget.NewCheck("Enable Google Cloud Backup", func(b bool) {
		localOptions.UseGoogleCloudBackup = b
	})
	googleCheckBox.Checked = localOptions.UseGoogleCloudBackup

	cloudDriverBox.Add(localFsBox)
	cloudDriverBox.Add(googleCheckBox)

	container.Add(cloudDriverBox)

	container.Add(widget.NewButton("Delete Cached Cloud Tokens", func() {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			fmt.Println(err)
			return
		}

		err = os.Remove(cacheDir + string(os.PathSeparator) + "token.json")
		if err != nil {
			fmt.Println(err)
		}
	}))

	container.Add(widget.NewButton("Close without Saving", func() {
		ReloadUserOptions()
		GetViewStack().PopContent()
	}))

	saveAndCloseButton := widget.NewButton("Save and Close", func() {
		CommitUserOptions()
		GetViewStack().PopContent()
	})
	saveAndCloseButton.Importance = widget.HighImportance
	container.Add(saveAndCloseButton)

	return container
}
