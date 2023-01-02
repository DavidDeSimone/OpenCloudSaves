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
	"time"

	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	"google.golang.org/api/drive/v3"
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

// // Retrieve a token, saves the token, then returns the generated client.
// func getClient(config *oauth2.Config) *http.Client {
// 	// The file token.json stores the user's access and refresh tokens, and is
// 	// created automatically when the authorization flow completes for the first
// 	// time.
// 	tokFile := "token.json"
// 	tok, err := tokenFromFile(tokFile)
// 	if err != nil {
// 		tok = getTokenFromWeb(config)
// 		saveToken(tokFile, tok)
// 	}
// 	return config.Client(context.Background(), tok)
// }

// // Request a token from the web, then returns the retrieved token.
// func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
// 	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
// 	listener, err := net.Listen("tcp", DEFAULT_PORT)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	openbrowser(authURL)

// 	var tok *oauth2.Token
// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte("Success! You can safely close this tab."))
// 		tok, err = config.Exchange(context.TODO(), r.FormValue("code"), oauth2.AccessTypeOffline)
// 		listener.Close()
// 	})

// 	http.Serve(listener, nil)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	return tok
// }

// // Retrieves a token from a local file.
// func tokenFromFile(file string) (*oauth2.Token, error) {
// 	f, err := os.Open(file)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer f.Close()
// 	tok := &oauth2.Token{}
// 	err = json.NewDecoder(f).Decode(tok)
// 	return tok, err
// }

// // Saves a token to a file path.
// func saveToken(path string, token *oauth2.Token) {
// 	fmt.Printf("Saving credential file to: %s\n", path)
// 	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
// 	if err != nil {
// 		log.Fatalf("Unable to cache oauth token: %v", err)
// 	}
// 	defer f.Close()
// 	json.NewEncoder(f).Encode(token)
// }

// func makeService() *drive.Service {
// 	ctx := context.Background()
// 	b, err := creds.ReadFile("credentials.json")
// 	if err != nil {
// 		log.Fatalf("Unable to read client secret file: %v", err)
// 	}

// 	// If modifying these scopes, delete your previously saved token.json.
// 	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
// 	config.Endpoint = google.Endpoint
// 	config.RedirectURL = fmt.Sprintf("http://localhost%v/", DEFAULT_PORT)
// 	if err != nil {
// 		log.Fatalf("Unable to parse client secret file to config: %v", err)
// 	}
// 	client := getClient(config)

// 	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		log.Fatalf("Unable to retrieve Drive client: %v", err)
// 	}

// 	return srv
// }

func validateAndCreateParentFolder(srv *drive.Service) string {
	r, err := srv.Files.List().
		Q("'root' in parents").
		Fields("nextPageToken, files(id, name)").
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	createSaveFolder := true
	var saveFolderId string
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {
		for _, i := range r.Files {
			if i.Name == SAVE_FOLDER {
				createSaveFolder = false
				saveFolderId = i.Id
			}
			fmt.Printf("%s (%s) (%s)\n", i.Name, i.Id, i.ModifiedByMeTime)
		}
	}

	if createSaveFolder {
		f := &drive.File{
			Name:     SAVE_FOLDER,
			MimeType: "application/vnd.google-apps.folder",
		}

		LogVerbose("Creating steamsaves folder....")
		x, err := srv.Files.Create(f).Do()
		saveFolderId = x.Id
		if err != nil {
			log.Fatal(err)
		}
	}

	return saveFolderId
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

func sync(srv *drive.Service, input chan SyncRequest, output chan SyncResponse) {
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

func createOperation(srv *drive.Service, request SyncRequest, output chan SyncResponse) {
	createFile(srv, request.ParentId, request.Name, request.Path, output)
}

func downloadOperation(srv *drive.Service, request SyncRequest, output chan SyncResponse) {
	err := downloadFile(srv, request.FileId, request.Name, request.Dryrun)
	output <- SyncResponse{
		Operation: Download,
		Name:      request.Name,
		Path:      request.Path,
		Err:       err,
	}
}

func uploadOperation(srv *drive.Service, request SyncRequest, output chan SyncResponse) {
	result, err := uploadFile(srv, request.FileId, request.Path, request.Dryrun)
	output <- SyncResponse{
		Operation: Download,
		Result:    result,
		Name:      request.Name,
		Path:      request.Path,
		Err:       err,
	}
}

func downloadFile(srv *drive.Service, fileId string, fileName string, dryrun bool) error {
	if dryrun {
		LogVerbose("Dry Run - Downloading ", fileName)
		return nil
	}

	fileref, err := srv.Files.Get(fileId).Fields("modifiedTime").Do()
	if err != nil {
		return err
	}

	LogVerbose("Downloading ", fileName)
	res, err := srv.Files.Get(fileId).Download()
	if err != nil {
		return err
	}

	defer res.Body.Close()
	osf, err := os.Create(fileName)
	if err != nil {
		return err
	}

	io.Copy(osf, res.Body)
	osf.Close()
	modtime, err := time.Parse(time.RFC3339, fileref.ModifiedTime)
	if err != nil {
		return err
	}

	err = os.Chtimes(fileName, modtime, modtime)
	// Since we are downloading, we do not need to update the file hash or modified time
	// in the meta file
	if err != nil {
		return err
	}

	return nil
}

func uploadFile(srv *drive.Service, fileId string, filePath string, dryrun bool) (string, error) {
	LogVerbose("Local file is newer... uploading... ", filePath)
	if dryrun {
		fmt.Println("Dry-Run Uploading File (not actually uploading) to remote: ", filePath)
		return "", nil
	}

	osf, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer osf.Close()
	stat, err := osf.Stat()
	if err != nil {
		return "", err
	}

	modifiedAtTime := stat.ModTime().Format(time.RFC3339)
	ff := &drive.File{
		ModifiedTime: modifiedAtTime,
	}

	res, err := srv.Files.Update(fileId, ff).Media(osf).Do()
	if err != nil {
		return "", err
	}

	LogVerbose("Successfully uploaded ", res.Name, ", last modified ", res.ModifiedTime, ", ", stat.ModTime().Format(time.RFC3339))
	return modifiedAtTime, nil
}

func createFile(srv *drive.Service, parentId string, fileName string, filePath string, syncChan chan SyncResponse) {
	file, err := os.Open(filePath)
	if err != nil {
		syncChan <- SyncResponse{
			Operation: Create,
			Err:       err,
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		syncChan <- SyncResponse{
			Operation: Create,
			Err:       err,
		}
		return
	}

	modtime := stat.ModTime().Format(time.RFC3339)
	saveUpload := &drive.File{
		Name:         fileName,
		ModifiedTime: modtime,
		Parents:      []string{parentId},
	}

	_, err = srv.Files.Create(saveUpload).Media(file).Do()
	if err != nil {
		syncChan <- SyncResponse{
			Operation: Create,
			Err:       err,
		}
		return
	}

	syncChan <- SyncResponse{
		Operation: Create,
		Name:      fileName,
		Path:      filePath,
		Result:    modtime,
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

func syncFiles(srv *drive.Service, parentId string, syncPath string, files map[string]SyncFile, dryrun bool) error {
	// Test if folder exists, and if it does, what it contains
	// Update folder with data if file names match and files are newer
	inputChannel := make(chan SyncRequest, 1000)
	outputChannel := make(chan SyncResponse, 1000)
	for i := 0; i < WORKER_POOL_SIZE; i++ {
		go sync(srv, inputChannel, outputChannel)
	}

	for k, v := range files {
		if v.IsDir {
			pid, err := getGameFileId(srv, parentId, k)
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

	LogVerbose("Querying from parent ", parentId)
	// 1. Query current files on cloud:
	r, err := srv.Files.List().
		Q(fmt.Sprintf("'%v' in parents", parentId)).
		Fields("nextPageToken, files(id, name)").
		Do()

	if err != nil {
		return err
	}

	clientuuid, err := getClientUUID()
	if err != nil {
		return err
	}

	var metadata *GameMetadata = nil
	metafileId := ""
	for _, file := range r.Files {
		LogVerbose("Looking for Metafile, examining ", file.Name)
		if file.Name == STEAM_METAFILE {
			metafileId = file.Id
			res, err := srv.Files.Get(file.Id).Download()
			if err != nil {
				return err
			}
			defer res.Body.Close()

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}

			metadata = &GameMetadata{}
			err = json.Unmarshal(bytes, metadata)
			if err != nil {
				return err
			}
			break
		}
	}

	mustCreateMetaFile := false
	if metadata == nil {
		mustCreateMetaFile = true
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
				for _, f := range r.Files {
					if f.Name == k {
						if dryrun {
							fmt.Printf("Dry Run - Would Delete %v on cloud (not really deleting\n", f.Name)
						} else {
							if localMetadata.Files[k].Sha256 != metadata.Files[k].Sha256 {
								// CONFLICT - the file that we plan on deleting is NOT the same as on the server
								// We should surface to the user if we want to delete this.
								fmt.Println("CONFLICT!!!! ")
							} else {
								LogVerbose("Deleting file ", f.Name)
								err = srv.Files.Delete(f.Id).Do()
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
	for _, file := range r.Files {
		LogVerbose("Examing ", file.Name)
		// @TODO this should be an extension valid check....
		if file.Name == STEAM_METAFILE {
			continue
		}

		_, deleted := deletedFiles[file.Name]
		if deleted {
			continue
		}

		syncfile, found := files[file.Name]
		fullpath := syncfile.Name
		newFileHash := ""
		if !found {
			// 2a. Not present on local file system, download...
			// downloadFile(srv, file.Id, syncPath+file.Name, dryrun)
			inputChannel <- SyncRequest{
				Operation: Download,
				FileId:    file.Id,
				Path:      syncPath + file.Name,
				Name:      file.Name,
				Dryrun:    dryrun,
			}
			pendingUploadDownload += 1
		} else {
			// 2b. Present on local file system, compare to remote if we will upload or download...
			meta, ok := metadata.Files[file.Name]
			if !ok {
				// @TODO handle this more gracefully?
				LogVerbose(fmt.Errorf("cloud upload with corrupt metadata entry for %s", file.Name))
				continue
			}

			f, err := os.Open(fullpath)
			if err != nil {
				return err
			}
			defer f.Close()

			localfile, err := f.Stat()
			if err != nil {
				return err
			}

			local_modtime := localfile.ModTime()
			remote_modtime, err := time.Parse(time.RFC3339, meta.LastModified)
			if err != nil {
				return err
			}

			newFileHash, err = getFileHash(fullpath)
			if err != nil {
				return err
			}

			LogVerbose("Comparing", file.Name, " (remote): ", meta.Sha256)
			if local_modtime.Equal(remote_modtime) || newFileHash == meta.Sha256 {
				fmt.Println("Remote and local files in sync (id/mod timestamp) ", file.Id)
			} else if local_modtime.After(remote_modtime) {
				inputChannel <- SyncRequest{
					Operation: Upload,
					FileId:    file.Id,
					Path:      fullpath,
					Name:      file.Name,
					Dryrun:    dryrun,
				}

				pendingUploadDownload += 1
			} else {
				inputChannel <- SyncRequest{
					Operation: Download,
					FileId:    file.Id,
					Path:      fullpath,
					Name:      file.Name,
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
		for _, f := range r.Files {
			if k == f.Name {
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
	metaUpload := &drive.File{
		Name: STEAM_METAFILE,
	}

	mf, err := os.Open(syncPath + STEAM_METAFILE)
	if err != nil {
		return err
	}

	defer mf.Close()

	if mustCreateMetaFile {
		metaUpload.Parents = []string{parentId}
		_, err = srv.Files.Create(metaUpload).Media(mf).Do()
		if err != nil {
			return err
		}
	} else {
		_, err = srv.Files.Update(metafileId, metaUpload).Do()
		if err != nil {
			return err
		}
	}

	return nil
}

func getGameFileId(srv *drive.Service, parentId string, name string) (string, error) {
	resultId := ""
	res, err := srv.Files.List().
		Q(fmt.Sprintf("'%v' in parents", parentId)).
		Fields("nextPageToken, files(id, name)").
		Do()

	if err != nil {
		fmt.Println("Failed to find file for (parent/name) ", parentId, name)
		return resultId, err
	}

	for _, i := range res.Files {
		if i.Name == name {
			LogVerbose("Found id for (parent/id)", parentId, i.Id)
			resultId = i.Id
			break
		}
	}

	if resultId == "" {
		f := &drive.File{
			Name:     name,
			Parents:  []string{parentId},
			MimeType: "application/vnd.google-apps.folder",
		}

		LogVerbose("Creating Folder for Game", name)
		result, err := srv.Files.Create(f).Do()
		if err != nil {
			return resultId, err
		}

		resultId = result.Id
	}

	LogVerbose("Identified game", name, "with id ", resultId)
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

	srv := makeService()
	saveFolderId := validateAndCreateParentFolder(srv)
	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		id, err := getGameFileId(srv, saveFolderId, gamename)
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

			parentId, err := getGameFileId(srv, id, syncpath.Parent)
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
