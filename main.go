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

//go:embed credentials.json
var creds embed.FS

const APP_NAME string = "SteamCustomCloudUpload"
const SAVE_FOLDER string = "steamsave"
const DEFAULT_PORT string = ":54438"
const STEAM_METAFILE string = "steamcloudloadmeta.json"
const CURRENT_META_VERSION int = 1
const WORKER_POOL_SIZE = 4

var verboseLogging bool = false

func LogVerbose(v ...any) {
	if verboseLogging {
		log.Println(v...)
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

func getClientUUID() (string, error) {
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
		LogVerbose("Generating new UUID for client...")
		result = uuid.New().String()
		err = os.WriteFile(fileName, []byte(result), os.ModePerm)
		if err != nil {
			return "", err
		}
	} else {
		result = string(f)
	}

	LogVerbose("UUID for client ", result)
	return result, nil
}

type FileMetadata struct {
	Sha256         string `json:"sha256"`
	LastModified   string `json:"lastmodified"`
	Lastclientuuid string `json:"lastclientuuid"`
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
	Err       error
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
	if result != nil {
		resultModtime = result.GetModTime()
	}

	output <- SyncResponse{
		Operation: Create,
		Name:      request.Name,
		Path:      request.Path,
		Result:    resultModtime,
		Err:       err,
	}
}

func downloadOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	err := srv.DownloadFile(request.FileId, request.Name) //downloadFile(srv, request.FileId, request.Name, request.Dryrun)
	output <- SyncResponse{
		Operation: Download,
		Name:      request.Name,
		Path:      request.Path,
		Err:       err,
	}
}

func uploadOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.UploadFile(request.FileId, request.Path, request.Name) //uploadFile(srv, request.FileId, request.Path, request.Dryrun)
	resultModtime := ""
	if result != nil {
		resultModtime = result.GetModTime()
	}

	output <- SyncResponse{
		Operation: Download,
		Result:    resultModtime,
		Name:      request.Name,
		Path:      request.Path,
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

func syncFiles(srv CloudDriver, parentId string, syncPath string, files map[string]SyncFile, dryrun bool) error {
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
			LogVerbose("Syncing Files (parent) ", parentId)
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

			err = syncFiles(srv, pid, parentPath, fileMap, dryrun)
			if err != nil {
				return err
			}
		}
	}

	clientuuid, err := getClientUUID()
	if err != nil {
		return err
	}
	LogVerbose("Querying from parent ", parentId)
	// 1. Query current files on cloud:
	r, err := srv.ListFiles(parentId)
	if err != nil {
		return err
	}

	metadata, err := srv.GetMetaData(parentId, STEAM_METAFILE)
	if metadata == nil {
		metadata = &GameMetadata{
			Version: CURRENT_META_VERSION,
			Gameid:  parentId,
			Files:   make(map[string]FileMetadata),
		}
	}

	localMetafile, err := os.ReadFile(syncPath + STEAM_METAFILE)
	var localMetadata *GameMetadata = nil
	if err == nil {
		// If we don't have a local metafile, that is fine.
		localMetadata = &GameMetadata{}
		err = json.Unmarshal(localMetafile, localMetadata)
		if err != nil {
			return err
		}
	}

	var deletedFiles map[string]bool = make(map[string]bool)
	// 3. Handle the case for deleting save data
	if localMetadata != nil {
		// If a file is in our local metafile, but not present locally, delete on cloud.
		for k := range localMetadata.Files {
			if _, err := os.Stat(syncPath + k); errors.Is(err, os.ErrNotExist) {
				fmt.Println("Path does not exist " + (syncPath + k))
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
								LogVerbose("Deleting file ", f.GetName())

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

	pendingUploadDownload := 0
	for _, file := range r {
		LogVerbose("Examing ", file.GetName())
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
				fmt.Println("Remote and local files in sync (id/mod timestamp) ", file.GetId())
			} else if fileSyncStatus == RemoteNewer {
				inputChannel <- SyncRequest{
					Operation: Upload,
					FileId:    file.GetId(),
					Path:      fullpath,
					Name:      file.GetName(),
					Dryrun:    dryrun,
				}

				pendingUploadDownload += 1
			} else {
				inputChannel <- SyncRequest{
					Operation: Download,
					FileId:    file.GetId(),
					Path:      fullpath,
					Name:      file.GetName(),
					Dryrun:    dryrun,
				}
				pendingUploadDownload += 1
			}
		}
	}

	for pendingUploadDownload > 0 {
		response := <-outputChannel
		if response.Err != nil {
			return response.Err
		}
		newModifiedTime := ""
		if response.Operation == Upload {
			newModifiedTime = response.Result
		}
		fullpath := response.Path
		fileName := response.Name

		newFileHash, err := getFileHash(fullpath)
		if err != nil {
			return err
		}

		current, ok := metadata.Files[fileName]

		if !ok {
			metadata.Files[fileName] = FileMetadata{
				Sha256:         newFileHash,
				LastModified:   newModifiedTime,
				Lastclientuuid: clientuuid,
			}
		} else {
			current.Lastclientuuid = clientuuid
			if newFileHash != "" {
				current.Sha256 = newFileHash
			}

			if newModifiedTime != "" {
				current.LastModified = newModifiedTime
			}
		}

		pendingUploadDownload -= 1
	}

	// 4. Check for local files not present on the cloud
	numCreations := 0
	for k, v := range files {
		found := false
		for _, f := range r {
			if k == f.GetName() {
				found = true
				LogVerbose("Found file ", k, " on cloud.")
				break
			}
		}

		if !found {
			if dryrun {
				fmt.Println("Dry-Run: Uploading File (not actually uploading): ", k)
			}
			LogVerbose("Uploading Initial File ", v)

			inputChannel <- SyncRequest{
				Operation: Create,
				ParentId:  parentId,
				Name:      k,
				Path:      v.Name,
			}
			numCreations += 1

		}
	}

	for numCreations > 0 {
		results := <-outputChannel

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
		}

		LogVerbose("Successfully uploaded save file ", results.Name)

		numCreations -= 1
	}

	metadata.Version = CURRENT_META_VERSION
	b, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	err = os.WriteFile(syncPath+STEAM_METAFILE, b, os.ModePerm)

	if err != nil {
		return nil
	}

	srv.UpdateMetaData(parentId, STEAM_METAFILE, syncPath+STEAM_METAFILE, metadata)
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
			LogVerbose("Found id for (parent/id)", parentId, i.GetId())
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

func CliMain(ops *Options, dm *GameDefManager) {
	verboseLogging = len(ops.Verbose) == 1 && ops.Verbose[0]
	dryrun := len(ops.DryRun) == 1 && ops.DryRun[0]
	LogVerbose("Verbose logging enabled...")

	addCustomGamesArgsLen := len(ops.AddCustomGames)
	if addCustomGamesArgsLen > 0 {
		for key, value := range ops.AddCustomGames {
			dm.AddUserOverride(key, value)
			LogVerbose("Added custom game... ", key)
		}

		return
	}

	srv := &GoogleCloudDriver{}
	srv.InitDriver()

	saveFolderId, err := validateAndCreateParentFolder(srv)
	if err != nil {
		log.Println(err)
		return
	}

	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		id, err := createDirIfNotExists(srv, saveFolderId, gamename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		syncpaths, err := dm.GetSyncpathForGame(gamename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, syncpath := range syncpaths {
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
			LogVerbose("Syncing Files (parent) ", parentId)
			err = syncFiles(srv, parentId, syncpath.Path, files, dryrun)
			if err != nil {
				fmt.Println(err)
				continue
			}
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
		CliMain(ops, dm)
	} else {
		GuiMain(ops, dm)
	}
}
