package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Gamenames      []string          `short:"g" long:"gamenames" description:"The name of the game(s) you will attempt to sync"`
	NoGUI          []bool            `short:"u" long:"no-gui" description:"Run in CLI mode with no GUI"`
	AddCustomGames map[string]string `short:"a" long:"add-custom-games" description:"<KEY>:<JSON_VALUE> Adds a custom game description to user_overrides.json. This accepts a JSON blobs in the format defined in gamedef_map.json"`
	UserOverride   []string          `short:"o" long:"user-override" description:"--user-override <FILE> Provide location for custom user override JSON file for game definitions"`
	PrintGameDefs  []bool            `short:"p" long:"print-gamedefs" description:"Print current gamedef map as JSON"`
}

type Message struct {
	Finished bool
	Message  string
	Err      error
}

type Cancellation struct {
	ShouldCancel bool
}

type ProgressEvent struct {
	Current int64
	Total   int64
}

type ChannelProvider struct {
	logs     chan Message
	cancel   chan Cancellation
	progress chan ProgressEvent
}

const CloudOperationDownload = 1 << 0
const CloudOperationUpload = 1 << 1
const CloudOperationDelete = 1 << 2
const CloudOperationAll = CloudOperationDownload | CloudOperationDelete | CloudOperationUpload

//go:embed credentials.json
var creds embed.FS

const APP_NAME = "OpenCloudSave"

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
			remotePath := fmt.Sprintf("%v%v/%v/", ToplevelCloudFolder, gamename, syncpath.Parent)
			LogMessage(logs, "Performing BiDirectional Sync: "+remotePath)

			//@TODO
			// This needs a way to respect ignore/exts
			// This should be as simple as --include/--exclude flags
			err := cm.BisyncDir(GetOneDriveStorage(), syncpath.Path, remotePath)
			if err != nil {
				fmt.Println(err)
				continue
			}

			LogMessage(logs, "All Operations Complete, files in sync")
			logs <- Message{
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
	cm := MakeCloudManager()
	err := cm.CreateDriveIfNotExists(GetOneDriveStorage())
	if err != nil {
		log.Fatal(err)
	}
	ops := &Options{}
	_, err = flags.Parse(ops)

	if err != nil {
		log.Fatal(err)
	}

	noGui := len(ops.NoGUI) == 1 && ops.NoGUI[0]
	userOverrideLocation := ""
	if len(ops.UserOverride) > 0 {
		userOverrideLocation = ops.UserOverride[0]
	}

	err = ApplyCloudUserOverride(cm, userOverrideLocation)
	if err != nil {
		log.Fatal(err)
	}

	dm := MakeGameDefManager(userOverrideLocation)
	dm.SetCloudManager(cm)
	dm.CommitUserOverrides()

	if noGui {
		channels := &ChannelProvider{
			logs:     make(chan Message, 100),
			progress: make(chan ProgressEvent, 15),
		}
		go consoleLogger(channels.logs)
		CliMain(cm, ops, dm, channels)
	} else {

		GuiMain(ops, dm)
	}
}
