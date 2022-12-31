package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jessevdk/go-flags"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Options struct {
	Verbose   []bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
	Gamenames []string `short:"g" long:"gamenames" description:"The name of the game(s) you will attempt to sync"`
	Gamepath  []string `short:"p" long:"gamepath" description:"The path to your game"`
	Sync      []bool   `short:"s" long:"sync" description:"Pull/Push from the server, with a prompt on conflict"`
	DryRun    []bool   `short:"d" long:"dry-run" description:"Run through the sync process without uploading/downloading from the cloud"`
	GUI       []bool   `short:"u" long:"gui" description:"Shows a GUI to manage cloud uploads (if available)"`
}

//go:embed credentials.json
var creds embed.FS

const APP_NAME string = "SteamCustomCloudUpload"
const SAVE_FOLDER string = "steamsave"
const DEFAULT_PORT string = ":54438"
const STEAM_METAFILE string = "steamcloudloadmeta.json"
const CURRENT_META_VERSION int = 1

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

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	listener, err := net.Listen("tcp", DEFAULT_PORT)
	if err != nil {
		log.Fatal(err)
	}

	openbrowser(authURL)

	var tok *oauth2.Token
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success! You can safely close this tab."))
		tok, err = config.Exchange(context.TODO(), r.FormValue("code"), oauth2.AccessTypeOffline)
		listener.Close()
	})

	http.Serve(listener, nil)
	if err != nil {
		fmt.Println(err)
	}

	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func makeService() *drive.Service {
	ctx := context.Background()
	b, err := creds.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	config.Endpoint = google.Endpoint
	config.RedirectURL = fmt.Sprintf("http://localhost%v/", DEFAULT_PORT)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	return srv
}

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

func syncFiles(srv *drive.Service, parentId string, syncPath string, files map[string]string, dryrun bool) error {
	// Test if folder exists, and if it does, what it contains
	// Update folder with data if file names match and files are newer

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

	for _, file := range r.Files {
		LogVerbose("Examing ", file.Name)
		// @TODO this should be an extension valid check....
		if file.Name == STEAM_METAFILE {
			continue
		}

		fullpath, found := files[file.Name]
		newFileHash := ""
		newModifiedTime := ""

		if !found {
			// 2a. Not present on local file system, download...
			if dryrun {
				LogVerbose("Dry Run - Downloading ", file.Name)
				continue
			}

			fileref, err := srv.Files.Get(file.Id).Fields("modifiedTime").Do()
			if err != nil {
				return err
			}

			LogVerbose("Downloading ", file.Name)
			res, err := srv.Files.Get(file.Id).Download()
			if err != nil {
				return err
			}

			defer res.Body.Close()
			osf, err := os.Create(syncPath + file.Name)
			if err != nil {
				return err
			}

			io.Copy(osf, res.Body)
			osf.Close()
			modtime, err := time.Parse(time.RFC3339, fileref.ModifiedTime)
			if err != nil {
				return err
			}

			err = os.Chtimes(syncPath+file.Name, modtime, modtime)
			// Since we are downloading, we do not need to update the file hash or modified time
			// in the meta file
			if err != nil {
				return err
			}
		} else {
			// 2b. Present on local file system, compare to remote if we will upload or download...
			meta, ok := metadata.Files[file.Name]
			if !ok {
				// @TODO handle this more gracefully?
				return fmt.Errorf("cloud upload with corrupt metadata entry for %s", file.Name)
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

			local_unix := local_modtime.UTC().Unix()
			remote_unix := remote_modtime.UTC().Unix()

			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				log.Fatal(err)
			}

			_, err = f.Seek(0, 0)
			if err != nil {
				return err
			}

			newFileHash = hex.EncodeToString(h.Sum(nil))

			LogVerbose("Comparing", file.Name, " (local/remote): ", local_unix, remote_unix, " ", newFileHash, ", ", meta.Sha256)
			if local_modtime.Equal(remote_modtime) || newFileHash == meta.Sha256 {
				fmt.Println("Remote and local files in sync (id/mod timestamp) ", file.Id, " ", local_unix)
			} else if local_modtime.After(remote_modtime) {
				LogVerbose("Local file is newer... uploading...")
				if dryrun {
					fmt.Println("Dry-Run Uploading File (not actually uploading) to remote: ", file.Name)
					continue
				}

				osf, err := os.Open(fullpath)
				if err != nil {
					return err
				}
				defer osf.Close()
				stat, err := osf.Stat()
				if err != nil {
					return err
				}

				modifiedAtTime := stat.ModTime().Format(time.RFC3339)
				ff := &drive.File{
					ModifiedTime: newModifiedTime,
				}

				res, err := srv.Files.Update(file.Id, ff).Media(osf).Do()
				if err != nil {
					return err
				}

				newModifiedTime = modifiedAtTime
				LogVerbose("Successfully uploaded ", res.Name, ", last modified ", res.ModifiedTime, ", ", stat.ModTime().Format(time.RFC3339))
			} else {
				// TODO better error handling around removal of save data
				if dryrun {
					fmt.Println("Dry-Run Downloading File (not actually downloading) from remote: ", file.Name)
					continue
				}

				res, err := srv.Files.Get(file.Id).Download()
				if err != nil {
					return err
				}

				defer res.Body.Close()
				err = os.Truncate(fullpath, 0)
				if err != nil {
					return err
				}

				osf, err := os.Open(fullpath)
				if err != nil {
					return err
				}

				io.Copy(osf, res.Body)
				osf.Close()
				err = os.Chtimes(fullpath, remote_modtime, remote_modtime)
				if err != nil {
					return err
				}

				LogVerbose("Successfully downloaded ", file.Name, ", last modified ", remote_modtime.Format(time.RFC3339))
			}
		}

		current, ok := metadata.Files[file.Name]
		if !ok {
			metadata.Files[file.Name] = FileMetadata{
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
	}

	// TODO
	// Handle the case for deleting save data

	// 3. Check for local files not present on the cloud
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
			osf, err := os.Open(v)
			if err != nil {
				return err
			}

			stat, err := osf.Stat()
			if err != nil {
				return err
			}

			h := sha256.New()
			if _, err := io.Copy(h, osf); err != nil {
				log.Fatal(err)
			}

			filehash := hex.EncodeToString(h.Sum(nil))

			_, err = osf.Seek(0, 0)
			if err != nil {
				return err
			}

			modtime := stat.ModTime().Format(time.RFC3339)
			saveUpload := &drive.File{
				Name:         k,
				ModifiedTime: modtime,
				Parents:      []string{parentId},
			}

			uploadedResult, err := srv.Files.Create(saveUpload).Media(osf).Do()
			if err != nil {
				log.Fatal(err)
			}

			osf.Close()
			metadata.Files[k] = FileMetadata{
				Sha256:         filehash,
				LastModified:   modtime,
				Lastclientuuid: clientuuid,
			}

			LogVerbose("Successfully uploaded save file ", uploadedResult.Name, "with id: ", uploadedResult.Id)
		}
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
	var resultId string
	res, err := srv.Files.List().
		Q(fmt.Sprintf("'%v' in parents", parentId)).
		Fields("nextPageToken, files(id, name)").
		Do()

	if err != nil || len(res.Files) == 0 {
		return resultId, err
	}

	for _, i := range res.Files {
		if i.Name == name {
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
	sync := len(ops.Sync) == 1 && ops.Sync[0]
	verboseLogging = len(ops.Verbose) == 1 && ops.Verbose[0]
	dryrun := len(ops.DryRun) == 1 && ops.DryRun[0]
	LogVerbose("Verbose logging enabled...")

	srv := makeService()
	saveFolderId := validateAndCreateParentFolder(srv)
	for _, gamename := range ops.Gamenames {
		gamename = strings.TrimSpace(gamename)
		id, err := getGameFileId(srv, saveFolderId, gamename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if sync {
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
				err = syncFiles(srv, parentId, syncpath.Path, files, dryrun)
				if err != nil {
					fmt.Println(err)
					continue
				}
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

	gui := len(ops.GUI) == 1 && ops.GUI[0]
	dm := MakeGameDefManager()

	if gui {
		GuiMain(ops, dm)
	} else {
		CliMain(ops, dm)
	}
}
