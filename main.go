package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strconv"

	"opencloudsave/core"
	"opencloudsave/gui"

	"github.com/jessevdk/go-flags"
)

func main() {
	if runtime.GOOS == "windows" {
		core.SetupWindowsConsole()
	}

	ops := &core.Options{}
	_, err := flags.Parse(ops)

	if err != nil {
		log.Fatal(err)
	}

	if len(ops.LogLocation) > 0 {
		err := core.InitLoggingWithPath(ops.LogLocation[0])
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := core.InitLoggingWithDefaultPath()
		if err != nil {
			log.Fatal(err)
		}
	}

	core.InfoLogger.Println("Launching with version " + core.VersionRevision)

	if len(ops.SetCloud) > 0 {
		cloud, err := strconv.Atoi(ops.SetCloud[0])
		if err != nil {
			log.Fatal(err)
		}

		cloudperfs := core.GetCurrentCloudPerfsOrDefault()
		cloudperfs.Cloud = cloud
		err = core.CommitCloudPerfs(cloudperfs)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Cloud Set!")
		return
	}

	storage, _ := core.GetCurrentCloudStorage()
	noGui := len(ops.NoGUI) == 1 && ops.NoGUI[0]
	userOverrideLocation := ""
	if len(ops.UserOverride) > 0 {
		userOverrideLocation = ops.UserOverride[0]
	}

	cm := core.MakeCloudManager()
	if len(ops.SyncUserSettings) > 0 && ops.SyncUserSettings[0] {
		if storage == nil {
			log.Fatal("Attempting to sync cloud data with no cloud provider set. Please set a cloud provider via --set-cloud <CLOUD_PROVIDER>")
		}

		err = cm.CreateDriveIfNotExists(context.Background(), storage)
		if err != nil {
			log.Fatal(err)
		}

		err = core.ApplyCloudUserOverride(cm, userOverrideLocation)
		if err != nil {
			core.ErrorLogger.Println(err)
			log.Fatal(err)
		}

		core.InfoLogger.Println("Cloud Settings In Sync")
		return
	}

	dm := core.MakeGameDefManager(userOverrideLocation)
	dm.SetCloudManager(cm)
	dm.CommitUserOverrides()

	if noGui {
		channels := core.MakeDefaultChannelProvider()
		go core.ConsoleLogger(channels.Logs)
		core.RequestMainOperation(context.Background(), cm, ops, dm, channels)
	} else {
		gui.GuiMain(ops, dm)
	}
}
