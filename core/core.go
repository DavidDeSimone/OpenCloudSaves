package core

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed version.txt
var VersionRevision string

type Options struct {
	Gamenames        []string          `short:"g" long:"gamenames" description:"The name of the game(s) you will attempt to sync"`
	NoGUI            []bool            `short:"u" long:"no-gui" description:"Run in CLI mode with no GUI"`
	AddCustomGames   map[string]string `short:"a" long:"add-custom-games" description:"<KEY>:<JSON_VALUE> Adds a custom game description to user_overrides.json. This accepts a JSON blobs in the format defined in gamedef_map.json"`
	UserOverride     []string          `short:"o" long:"user-override" description:"--user-override <FILE> Provide location for custom user override JSON file for game definitions"`
	PrintGameDefs    []bool            `short:"p" long:"print-gamedefs" description:"Print current gamedef map as JSON"`
	SyncUserSettings []bool            `short:"s" long:"sync-user-settings" description:"Attempt to sync user settings from the current cloud provider. If no cloud provider is set, will be a NO-OP."`
	SetCloud         []string          `short:"c" long:"set-cloud" description:"Sets the current cloud. 0 - GOOGLE, 1 - ONEDRIVE, 2 - DROPBOX, 3 - BOX, 4 - NEXTCLOUD, 5 - FTP"`
	DryRun           []bool            `short:"d" long:"dry-run" description:"Does not actually perform any network operations."`
	Verbose          []bool            `short:"v" long:"verbose" description:"Enable verbose logging"`
	LogLocation      []string          `short:"l" long:"log-location" description:"Specifies path to logfile. Defaults to User's Cache Dir / opencloudsave.log"`
	Experimental     []bool            `short:"e" long:"experimental" description:"E"`
}

type Message struct {
	Finished bool
	Message  string
	Err      error
}

type ChannelProvider struct {
	Logs   chan Message
	Cancel context.CancelFunc
}

const APP_NAME = "OpenCloudSave"

func MakeDefaultChannelProvider() *ChannelProvider {
	return &ChannelProvider{
		Logs:   make(chan Message, 100),
		Cancel: nil,
	}
}

func MakeChannelProviderWithCancelFunction(cancelFn context.CancelFunc) *ChannelProvider {
	provider := MakeDefaultChannelProvider()
	provider.Cancel = cancelFn
	return provider
}

func GetCurrentStorageProvider() Storage {
	storage, err := GetCurrentCloudStorage()
	if err != nil {
		ErrorLogger.Println(err)
		return nil
	}
	return storage
}

func LogMessage(logs chan Message, format string, msg ...any) {
	logs <- Message{
		Message: fmt.Sprintf(format, msg...),
	}
}

func RequestMainOperation(ctx context.Context, cm *CloudManager, ops *Options, dm GameDefManager, channels *ChannelProvider) {
	logs := channels.Logs

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

	gamedefs := dm.GetGameDefMap()
	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		gamedef := gamedefs[gamename]
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
			LogMessage(logs, "Performing Sync: "+remotePath)

			syncops := GetDefaultCloudOptions()
			if len(ops.DryRun) > 0 && ops.DryRun[0] {
				syncops.DryRun = true
			}

			if len(ops.Verbose) > 0 && ops.Verbose[0] {
				syncops.Verbose = true
			}

			syncops.CustomFlags = gamedef.CustomFlags
			syncops.Include = syncpath.Include

			result, err := cm.PerformSyncOperation(ctx, storage, syncops, syncpath.Path, remotePath)
			if err != nil {
				ErrorLogger.Println(err)
				logs <- Message{
					Err:      err,
					Finished: true,
				}
				continue
			}

			LogMessage(logs, "All Operations Complete")
			logs <- Message{
				Message:  result,
				Finished: true,
			}
		}
	}
}

func ConsoleLogger(input chan Message) {
	for {
		result := <-input
		if result.Finished {
			break
		}

		if result.Err != nil {
			ErrorLogger.Println(result.Err)
		} else {
			InfoLogger.Println(result.Message)
		}
	}
}
