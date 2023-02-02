package main

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Gamenames        []string          `short:"g" long:"gamenames" description:"The name of the game(s) you will attempt to sync"`
	NoGUI            []bool            `short:"u" long:"no-gui" description:"Run in CLI mode with no GUI"`
	AddCustomGames   map[string]string `short:"a" long:"add-custom-games" description:"<KEY>:<JSON_VALUE> Adds a custom game description to user_overrides.json. This accepts a JSON blobs in the format defined in gamedef_map.json"`
	UserOverride     []string          `short:"o" long:"user-override" description:"--user-override <FILE> Provide location for custom user override JSON file for game definitions"`
	PrintGameDefs    []bool            `short:"p" long:"print-gamedefs" description:"Print current gamedef map as JSON"`
	SyncUserSettings []bool            `short:"s" long:"--sync-user-settings" description:"Attempt to sync user settings from the current cloud provider. If no cloud provider is set, will be a NO-OP."`
	SetCloud         []string          `short:"c" long:"--set-cloud" description:"Sets the current cloud. 0 - GOOGLE, 1 - ONEDRIVE, 2 - DROPBOX, 3 - BOX, 4 - NEXTCLOUD"`
	DryRun           []bool            `short:"d" long:"--dry-run" description:"Does not actually perform any network operations."`
	Verbose          []bool            `short:"v" long:"--verbose" description:"Enable verbose logging"`
}

type Message struct {
	Finished bool
	Message  string
	Err      error
}

type ChannelProvider struct {
	logs chan Message
}

const APP_NAME = "OpenCloudSave"

func GetCurrentStorageProvider() Storage {
	storage, err := GetCurrentCloudStorage()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return storage
}

func LogMessage(logs chan Message, format string, msg ...any) {
	logs <- Message{
		Message: fmt.Sprintf(format, msg...),
	}
}

func CliMain(cm *CloudManager, ops *Options, dm GameDefManager, channels *ChannelProvider) {
	logs := channels.logs

	if len(ops.PrintGameDefs) > 0 {
		result := dm.GetGameDefMap()
		marshedResult, err := json.Marshal(result)
		fmt.Println(string(marshedResult))
		if err != nil {
			logs <- Message{
				Err:      err,
				Finished: true,
			}
		} else {
			logs <- Message{
				Message:  string(marshedResult),
				Finished: true,
			}
		}

		return
	}

	LogMessage(logs, "Main Initalized")

	addCustomGamesArgsLen := len(ops.AddCustomGames)
	if addCustomGamesArgsLen > 0 {
		for key, value := range ops.AddCustomGames {
			dm.AddUserOverride(key, value)
		}

		return
	}

	storage := GetCurrentStorageProvider()
	if storage == nil {
		logs <- Message{
			Finished: true,
			Err:      fmt.Errorf("no cloud provider set"),
		}

		return
	}

	LogMessage(logs, "Starting Upload Process...")

	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		LogMessage(logs, "Performing Check on %v", gamename)

		syncpaths, err := dm.GetSyncpathForGame(gamename)
		LogMessage(logs, "Identified Paths for %v: %v", gamename, syncpaths)
		if err != nil {
			fmt.Println(err)
			logs <- Message{
				Err: err,
			}
			continue
		}

		for _, syncpath := range syncpaths {
			LogMessage(logs, "Examining Path %v", syncpath.Path)
			remotePath := fmt.Sprintf("%v%v/", ToplevelCloudFolder, gamename)
			LogMessage(logs, "Performing BiDirectional Sync: "+remotePath)

			//@TODO
			// This needs a way to respect ignore/exts
			// This should be as simple as --include/--exclude flags
			syncops := GetDefaultCloudOptions()
			if len(ops.DryRun) > 0 && ops.DryRun[0] {
				syncops.DryRun = true
			}

			if len(ops.Verbose) > 0 && ops.Verbose[0] {
				syncops.Verbose = true
			}

			syncops.Include = syncpath.Include

			result, err := cm.BisyncDir(storage, syncops, syncpath.Path, remotePath)
			if err != nil {
				fmt.Println(err)
				continue
			}

			LogMessage(logs, "All Operations Complete, files in sync")
			logs <- Message{
				Message:  result,
				Finished: true,
			}
		}
	}
}

func consoleLogger(input chan Message) {
	for {
		result := <-input
		if result.Finished {
			break
		}

		if result.Err != nil {
			fmt.Println(result.Err)
		} else {
			fmt.Println(result.Message)
		}
	}
}

func main() {
	if runtime.GOOS == "windows" {
		SetupWindowsConsole()
	}

	ops := &Options{}
	_, err := flags.Parse(ops)

	if err != nil {
		log.Fatal(err)
	}

	if len(ops.SetCloud) > 0 {
		cloud, err := strconv.Atoi(ops.SetCloud[0])
		if err != nil {
			log.Fatal(err)
		}

		if cloud < GOOGLE || cloud > NEXT {
			log.Fatal("Invalid cloud")
		}

		cloudperfs := GetCurrentCloudPerfsOrDefault()
		cloudperfs.Cloud = cloud
		err = CommitCloudPerfs(cloudperfs)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Cloud Set!")
		return
	}

	storage, _ := GetCurrentCloudStorage()
	noGui := len(ops.NoGUI) == 1 && ops.NoGUI[0]
	userOverrideLocation := ""
	if len(ops.UserOverride) > 0 {
		userOverrideLocation = ops.UserOverride[0]
	}

	cm := MakeCloudManager()
	if len(ops.SyncUserSettings) > 0 && ops.SyncUserSettings[0] {
		if storage == nil {
			log.Fatal("Attempting to sync cloud data with no cloud provider set. Please set a cloud provider via --set-cloud <CLOUD_PROVIDER>")
		}

		err = cm.CreateDriveIfNotExists(storage)
		if err != nil {
			log.Fatal(err)
		}

		err = ApplyCloudUserOverride(cm, userOverrideLocation)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}

		fmt.Println("Cloud Settings In Sync")
		return
	}

	dm := MakeGameDefManager(userOverrideLocation)
	dm.SetCloudManager(cm)
	dm.CommitUserOverrides()

	if noGui {
		channels := &ChannelProvider{
			logs: make(chan Message, 100),
		}
		go consoleLogger(channels.logs)
		CliMain(cm, ops, dm, channels)
	} else {

		GuiMain(ops, dm)
	}
}
