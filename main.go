package main

import (
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Verbose        []bool            `short:"v" long:"verbose" description:"Show verbose debug information"`
	Gamenames      []string          `short:"g" long:"gamenames" description:"The name of the game(s) you will attempt to sync"`
	Gamepath       []string          `short:"p" long:"gamepath" description:"The path to your game"`
	DryRun         []bool            `short:"d" long:"dry-run" description:"Run through the sync process without uploading/downloading from the cloud"`
	NoGUI          []bool            `short:"u" long:"no-gui" description:"Run in CLI mode with no GUI"`
	AddCustomGames map[string]string `short:"a" long:"add-custom-games" description:"<KEY>:<JSON_VALUE> Adds a custom game description to user_overrides.json. This accepts a JSON blobs in the format defined in gamedef_map.json"`
	UserOverride   []string          `short:"o" long:"user-override" description:"--user-override <FILE> Provide location for custom user override JSON file for game definitions"`
}

type Message struct {
	Finished bool
	Message  string
	Err      error
}

type Cancellation struct {
	ShouldCancel bool
}

type ChannelProvider struct {
	logs   chan Message
	cancel chan Cancellation
}

const CloudOperationDownload = 1 << 0
const CloudOperationUpload = 1 << 1
const CloudOperationDelete = 1 << 2
const CloudOperationAll = CloudOperationDownload | CloudOperationDelete | CloudOperationUpload

//go:embed credentials.json
var creds embed.FS

const APP_NAME = "SteamCustomCloudUpload"
const SAVE_FOLDER = "steamsave"
const DEFAULT_PORT = ":54438"
const STEAM_METAFILE = "steamcloudloadmeta.json"
const CURRENT_META_VERSION = 1
const WORKER_POOL_SIZE = 12

var service CloudDriver = nil

func GetDefaultService() CloudDriver {
	if service == nil {
		service = &GoogleCloudDriver{}
		// service = &LocalFsCloudDriver{}
		service.InitDriver()

	}

	return service
}

func LogMessage(logs chan Message, format string, msg ...any) {
	logs <- Message{
		Message: fmt.Sprintf(format, msg...),
	}
}

func CliMain(ops *Options, dm GameDefManager, channels *ChannelProvider, localfs LocalFs) {
	logs := channels.logs
	cancel := channels.cancel

	// verboseLogging = len(ops.Verbose) == 1 && ops.Verbose[0]
	dryrun := len(ops.DryRun) == 1 && ops.DryRun[0]

	LogMessage(logs, "Main Initalized")

	addCustomGamesArgsLen := len(ops.AddCustomGames)
	if addCustomGamesArgsLen > 0 {
		for key, value := range ops.AddCustomGames {
			dm.AddUserOverride(key, value)
		}

		return
	}

	LogMessage(logs, "Starting Upload Process...")

	srv := GetDefaultService()
	saveFolderId, err := ValidateAndCreateParentFolder(srv)
	if err != nil {
		log.Println(err)
		return
	}

	LogMessage(logs, "Cloud Service Initialized...")

	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		LogMessage(logs, "Performing Check on %v", gamename)
		id, err := CreateRemoteDirIfNotExists(srv, saveFolderId, gamename)
		if err != nil {
			fmt.Println(err)
			logs <- Message{
				Err: err,
			}
			continue
		}

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
			files, err := dm.GetFilesForGame(gamename, syncpath.Parent)
			if err != nil {
				fmt.Println(err)
				logs <- Message{
					Err: err,
				}
				continue
			}

			parentId, err := CreateRemoteDirIfNotExists(srv, id, syncpath.Parent)
			if err != nil {
				fmt.Println(err)
				logs <- Message{
					Err: err,
				}
				continue
			}
			err = SyncFiles(srv, parentId, syncpath, files, dryrun, localfs, logs, cancel)
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
			fmt.Println("Console Logger Complete...")
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
	ops := &Options{}
	_, err := flags.Parse(ops)

	if err != nil {
		log.Fatal(err)
	}

	noGui := len(ops.NoGUI) == 1 && ops.NoGUI[0]
	userOverrideLocation := ""
	if len(ops.UserOverride) > 0 {
		userOverrideLocation = ops.UserOverride[0]
	}
	dm := MakeGameDefManager(userOverrideLocation)

	if noGui {
		channels := &ChannelProvider{
			logs:   make(chan Message, 100),
			cancel: make(chan Cancellation, 1),
		}

		go consoleLogger(channels.logs)
		CliMain(ops, dm, channels, GetDefaultLocalFs())
	} else {
		GuiMain(ops, dm)
	}
}
