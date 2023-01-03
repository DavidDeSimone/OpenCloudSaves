package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Verbose        []bool            `short:"v" long:"verbose" description:"Show verbose debug information"`
	Gamenames      []string          `short:"g" long:"gamenames" description:"The name of the game(s) you will attempt to sync"`
	Gamepath       []string          `short:"p" long:"gamepath" description:"The path to your game"`
	DryRun         []bool            `short:"d" long:"dry-run" description:"Run through the sync process without uploading/downloading from the cloud"`
	NoGUI          []bool            `short:"u" long:"no-gui" description:"Run in CLI mode with no GUI"`
	AddCustomGames map[string]string `short:"a" long:"add-custom-games" description:"<KEY>:<JSON_VALUE> Adds a custom game description to user_overrides.json. This accepts a JSON blobs in the format defined in gamedef_map.json"`
}

type FileMetadata struct {
	Sha256         string `json:"sha256"`
	LastModified   string `json:"lastmodified"`
	Lastclientuuid string `json:"lastclientuuid"`
	FileId         string `json:"fileid"`
}

type GameMetadata struct {
	Version int                     `json:"version"`
	Gameid  string                  `json:"gameid"`
	Files   map[string]FileMetadata `json:"files"`
	fileId  string
}

const (
	Create = iota
	Download
	Upload
)

type SyncRequest struct {
	Operation int
	Name      string
	Path      string
	ParentId  string
	FileId    string
	Dryrun    bool
}

type SyncResponse struct {
	Operation int
	Result    string
	Name      string
	Path      string
	FileId    string
	Err       error
}

type Message struct {
	Finished bool
	Message  string
	Err      error
}

//go:embed credentials.json
var creds embed.FS

const APP_NAME string = "SteamCustomCloudUpload"
const SAVE_FOLDER string = "steamsave"
const DEFAULT_PORT string = ":54438"
const STEAM_METAFILE string = "steamcloudloadmeta.json"
const CURRENT_META_VERSION int = 1
const WORKER_POOL_SIZE = 4

var verboseLogging bool = false

var service CloudDriver = nil

func GetDefaultService() CloudDriver {
	if service == nil {
		service = &GoogleCloudDriver{}
		service.InitDriver()

	}

	return service
}

// func LogVerbose(v ...any) {
// 	if verboseLogging {
// 		log.Println(v...)
// 	}
// }

func LogMessage(logs chan Message, format string, msg ...any) {
	logs <- Message{
		Message: fmt.Sprintf(format, msg...),
	}
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func validateAndCreateParentFolder(srv CloudDriver) (string, error) {
	files, err := srv.ListFiles("root")
	if err != nil {
		return "", err
	}

	createSaveFolder := true
	saveFolderFileId := ""
	for _, f := range files {
		if f.GetName() == SAVE_FOLDER {
			createSaveFolder = false
			saveFolderFileId = f.GetId()
			break
		}
	}

	if createSaveFolder {
		result, err := srv.CreateDir(SAVE_FOLDER, "root")
		if err != nil {
			return "", err
		}
		saveFolderFileId = result.GetId()
	}

	return saveFolderFileId, nil
}

var clientUUID string

func getClientUUID() (string, error) {
	if clientUUID != "" {
		return clientUUID, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	separator := string(os.PathSeparator)

	fileName := cacheDir + separator + APP_NAME + separator + "uuid"
	err = os.MkdirAll(cacheDir+separator+APP_NAME, os.ModePerm)
	if err != nil {
		return "", err
	}

	f, err := os.ReadFile(fileName)

	result := ""
	if err != nil {
		result = uuid.New().String()
		err = os.WriteFile(fileName, []byte(result), os.ModePerm)
		if err != nil {
			return "", err
		}
	} else {
		result = string(f)
	}

	clientUUID = result
	return result, nil
}

func sync(srv CloudDriver, input chan SyncRequest, output chan SyncResponse) {
	for {
		request := <-input
		switch request.Operation {
		case Create:
			createOperation(srv, request, output)
		case Download:
			downloadOperation(srv, request, output)
		case Upload:
			uploadOperation(srv, request, output)
		}
	}
}

func createOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.CreateFile(request.ParentId, request.Name, request.Path)
	resultModtime := ""
	resultFileId := ""
	if result != nil {
		resultModtime = result.GetModTime()
		resultFileId = result.GetId()
	}

	output <- SyncResponse{
		Operation: Create,
		Name:      request.Name,
		Path:      request.Path,
		Result:    resultModtime,
		FileId:    resultFileId,
		Err:       err,
	}
}

func downloadOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.DownloadFile(request.FileId, request.Path, request.Name) //downloadFile(srv, request.FileId, request.Name, request.Dryrun)
	resultModtime := ""
	resultFileId := ""
	if result != nil {
		resultModtime = result.GetModTime()
		resultFileId = result.GetId()
	}

	output <- SyncResponse{
		Operation: Download,
		Name:      request.Name,
		Path:      request.Path,
		FileId:    resultFileId,
		Result:    resultModtime,
		Err:       err,
	}
}

func uploadOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.UploadFile(request.FileId, request.Path, request.Name) //uploadFile(srv, request.FileId, request.Path, request.Dryrun)
	resultModtime := ""
	resultFileId := ""
	if result != nil {
		resultModtime = result.GetModTime()
		resultFileId = result.GetId()
	}

	output <- SyncResponse{
		Operation: Download,
		Result:    resultModtime,
		Name:      request.Name,
		Path:      request.Path,
		FileId:    resultFileId,
		Err:       err,
	}
}

func getFileHash(fileName string) (string, error) {
	f, err := os.Open(fileName)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func GetLocalMetadata(filePath string) (*GameMetadata, error) {
	localMetafile, err := os.ReadFile(filePath)
	var localMetadata *GameMetadata = nil
	if err == nil {
		// If we don't have a local metafile, that is fine.
		localMetadata = &GameMetadata{}
		err = json.Unmarshal(localMetafile, localMetadata)
		if err != nil {
			return nil, err
		}
	}

	return localMetadata, nil
}

func syncFiles(srv CloudDriver, parentId string, syncPath string, files map[string]SyncFile, dryrun bool, logs chan Message) error {
	LogMessage(logs, "Syncing Files for %v", syncPath)

	// Test if folder exists, and if it does, what it contains
	// Update folder with data if file names match and files are newer
	inputChannel := make(chan SyncRequest, 1000)
	outputChannel := make(chan SyncResponse, 1000)
	for i := 0; i < WORKER_POOL_SIZE; i++ {
		go sync(srv, inputChannel, outputChannel)
	}

	for k, v := range files {
		if v.IsDir {
			pid, err := createDirIfNotExists(srv, parentId, k)
			if err != nil {
				return err
			}
			separator := string(os.PathSeparator)
			parentPath := syncPath + separator + k + separator
			var fileMap map[string]SyncFile = make(map[string]SyncFile)
			f, err := os.Open(syncPath + separator + k + separator)
			if err != nil {
				return err
			}

			defer f.Close()
			files, err := f.Readdir(0)
			if err != nil {
				return err
			}

			for _, file := range files {
				isDir := false
				if file.IsDir() {
					isDir = true
				}

				fileMap[file.Name()] = SyncFile{
					Name:  parentPath + file.Name(),
					IsDir: isDir,
				}
			}

			err = syncFiles(srv, pid, parentPath, fileMap, dryrun, logs)
			if err != nil {
				return err
			}
		}
	}

	clientuuid, err := getClientUUID()
	LogMessage(logs, "Identified Client UUID (%v)", clientuuid)
	if err != nil {
		return err
	}
	// 1. Query current files on cloud:
	r, err := srv.ListFiles(parentId)
	if err != nil {
		return err
	}

	metadata, err := srv.GetMetaData(parentId, STEAM_METAFILE)
	if metadata == nil {
		LogMessage(logs, "Did not find remote metafile, initalizing... %v", parentId)
		metadata = &GameMetadata{
			Version: CURRENT_META_VERSION,
			Gameid:  parentId,
			Files:   make(map[string]FileMetadata),
		}
	}

	localMetadata, err := GetLocalMetadata(syncPath + STEAM_METAFILE)
	if err != nil {
		return err
	}

	LogMessage(logs, "-------- STAGE 1 -----------")
	LogMessage(logs, "Examining files present on cloud but deleted locally")
	var deletedFiles map[string]bool = make(map[string]bool)
	// 1. Handle the case for deleting save data
	if localMetadata != nil {
		// If a file is in our local metafile, but not present locally, delete on cloud.
		for k := range localMetadata.Files {
			if _, err := os.Stat(syncPath + k); errors.Is(err, os.ErrNotExist) {
				for _, f := range r {
					if f.GetName() == k {
						if dryrun {
							fmt.Printf("Dry Run - Would Delete %v on cloud (not really deleting\n", f.GetName())
						} else {
							if localMetadata.Files[k].Sha256 != metadata.Files[k].Sha256 {
								// CONFLICT - the file that we plan on deleting is NOT the same as on the server
								// We should surface to the user if we want to delete this.
								fmt.Println("CONFLICT!!!! ")
							} else {
								LogMessage(logs, "Deleting File (id) %v (%v)", f.GetName(), f.GetId())
								err = srv.DeleteFile(f.GetId())
								if err != nil {
									return err
								}
								delete(metadata.Files, k)
								deletedFiles[k] = true
							}
						}

						break
					}
				}
			}
		}
	}

	LogMessage(logs, "-------- STAGE 2 -----------")
	LogMessage(logs, "Downloading updates to exisiting files; Uploading exisiting files")
	pendingUploadDownload := 0
	for _, file := range r {
		// @TODO this should be an extension valid check....
		if file.GetName() == STEAM_METAFILE {
			continue
		}

		_, deleted := deletedFiles[file.GetName()]
		if deleted {
			continue
		}

		syncfile, found := files[file.GetName()]
		fullpath := syncfile.Name
		if !found {
			// 2a. Not present on local file system, download...
			// downloadFile(srv, file.Id, syncPath+file.GetName(), dryrun)
			LogMessage(logs, "Queued Download for %v", file.GetName())
			inputChannel <- SyncRequest{
				Operation: Download,
				FileId:    file.GetId(),
				Path:      syncPath + file.GetName(),
				Name:      file.GetName(),
				Dryrun:    dryrun,
			}
			pendingUploadDownload += 1
		} else {
			// 2b. Present on local file system, compare to remote if we will upload or download...
			fileSyncStatus, err := srv.IsFileInSync(file.GetName(), fullpath, file.GetId(), metadata)
			if err != nil {
				return err
			}

			if fileSyncStatus == InSync {
				LogMessage(logs, "Remote and local files in sync (id/mod timestamp) %v", file.GetId())
			} else if fileSyncStatus == RemoteNewer {
				inputChannel <- SyncRequest{
					Operation: Download,
					FileId:    file.GetId(),
					Path:      fullpath,
					Name:      file.GetName(),
					Dryrun:    dryrun,
				}

				LogMessage(logs, "Queued Download for %v", file.GetName())

				pendingUploadDownload += 1
			} else {
				inputChannel <- SyncRequest{
					Operation: Upload,
					FileId:    file.GetId(),
					Path:      fullpath,
					Name:      file.GetName(),
					Dryrun:    dryrun,
				}

				LogMessage(logs, "Queued Upload for %v", file.GetName())

				pendingUploadDownload += 1
			}
		}
	}

	for pendingUploadDownload > 0 {
		response := <-outputChannel
		if response.Err != nil {
			return response.Err
		}

		LogMessage(logs, "Operation complete for %v", response.Name)
		newModifiedTime := response.Result
		fullpath := response.Path
		fileName := response.Name

		newFileHash, err := getFileHash(fullpath)
		if err != nil {
			return err
		}

		metadata.Files[fileName] = FileMetadata{
			Sha256:         newFileHash,
			LastModified:   newModifiedTime,
			Lastclientuuid: clientuuid,
			FileId:         response.FileId,
		}

		pendingUploadDownload -= 1
	}

	LogMessage(logs, "-------- STAGE 3 -----------")
	LogMessage(logs, "Download new files from remote")
	// 3. Check for local files not present on the cloud
	numCreations := 0
	for k, v := range files {
		found := false
		for _, f := range r {
			if k == f.GetName() {
				found = true
				break
			}
		}

		if !found {
			if dryrun {
				fmt.Println("Dry-Run: Uploading File (not actually uploading): ", k)
				continue
			}

			inputChannel <- SyncRequest{
				Operation: Create,
				ParentId:  parentId,
				Name:      k,
				Path:      v.Name,
			}
			numCreations += 1
			LogMessage(logs, "Queue upload for file %v", v.Name)
		}
	}

	for numCreations > 0 {
		results := <-outputChannel
		LogMessage(logs, "Operation Successful for %v", results.Name)
		if results.Err != nil {
			return results.Err
		}

		filehash, err := getFileHash(results.Path)
		if err != nil {
			return err
		}

		metadata.Files[results.Name] = FileMetadata{
			Sha256:         filehash,
			LastModified:   results.Result,
			Lastclientuuid: clientuuid,
			FileId:         results.FileId,
		}

		numCreations -= 1
	}

	LogMessage(logs, "Data Upload/Download success - updating metadata...")
	metadata.Version = CURRENT_META_VERSION
	b, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	err = os.WriteFile(syncPath+STEAM_METAFILE, b, os.ModePerm)

	if err != nil {
		return nil
	}

	err = srv.UpdateMetaData(parentId, STEAM_METAFILE, syncPath+STEAM_METAFILE, metadata)
	if err != nil {
		return err
	}

	LogMessage(logs, "All Operations Complete, files in sync")
	logs <- Message{
		Finished: true,
	}

	return nil
}

func createDirIfNotExists(srv CloudDriver, parentId string, name string) (string, error) {
	resultId := ""
	res, err := srv.ListFiles(parentId)

	if err != nil {
		fmt.Println("Failed to find file for (parent/name) ", parentId, name)
		return resultId, err
	}

	for _, i := range res {
		if i.GetName() == name {
			resultId = i.GetId()
			break
		}
	}

	if resultId == "" {
		result, err := srv.CreateDir(name, parentId)
		if err != nil {
			return resultId, err
		}

		resultId = result.GetId()
	}

	return resultId, nil
}

func CliMain(ops *Options, dm *GameDefManager, logs chan Message) {
	verboseLogging = len(ops.Verbose) == 1 && ops.Verbose[0]
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
	saveFolderId, err := validateAndCreateParentFolder(srv)
	if err != nil {
		log.Println(err)
		return
	}

	LogMessage(logs, "Cloud Service Initialized...")

	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		LogMessage(logs, "Performing Check on %v", gamename)
		id, err := createDirIfNotExists(srv, saveFolderId, gamename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		syncpaths, err := dm.GetSyncpathForGame(gamename)
		LogMessage(logs, "Identified Paths for %v: %v", gamename, syncpaths)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, syncpath := range syncpaths {
			LogMessage(logs, "Examining Path %v", syncpath.Path)
			files, err := dm.GetFilesForGame(gamename, syncpath.Parent)
			if err != nil {
				fmt.Println(err)
				continue
			}

			parentId, err := createDirIfNotExists(srv, id, syncpath.Parent)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = syncFiles(srv, parentId, syncpath.Path, files, dryrun, logs)
			if err != nil {
				fmt.Println(err)
				continue
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
	dm := MakeGameDefManager()
	dm.ApplyUserOverrides()

	if noGui {
		logs := make(chan Message, 100)
		go consoleLogger(logs)
		CliMain(ops, dm, logs)
	} else {
		GuiMain(ops, dm)
	}
}
