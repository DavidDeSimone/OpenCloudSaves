package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Options struct {
	Verbose  []bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
	Gamename string   `short:"g" long:"gamename" description:"The name of the game you will attempt to sync"`
	Gamepath []string `short:"p" long:"gamepath" description:"The path to your game"`
	Sync     []bool   `short:"s" long:"sync" description:"Pull/Push from the server, with a prompt on conflict"`
	DryRun   []bool   `short:"d" long:"dry-run" description:"Run through the sync process without uploading/downloading from the cloud"`
}

const SAVE_FOLDER string = "steamsave"

var verboseLogging bool = false

func LogVerbose(v ...any) {
	if verboseLogging {
		log.Println(v...)
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
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
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
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
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
	fmt.Println("Files:")
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

func syncFiles(parentId string, syncPath string, files []string) error {
	// Test if folder exists, and if it does, what it contains
	// Update folder with data if file names match and files are newer
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

func main() {
	ops := &Options{}
	_, err := flags.Parse(ops)

	if err != nil {
		log.Fatal(err)
	}

	gamename := strings.TrimSpace(ops.Gamename)
	sync := len(ops.Sync) == 1 && ops.Sync[0]
	verboseLogging = len(ops.Verbose) == 1 && ops.Verbose[0]
	LogVerbose("Verbose logging enabled...")

	dm := MakeDriverManager()
	srv := makeService()
	saveFolderId := validateAndCreateParentFolder(srv)
	id, err := getGameFileId(srv, saveFolderId, gamename)
	if err != nil {
		log.Fatal(err)
	}

	if sync {
		files, err := dm.GetFilesForGame(gamename)
		if err != nil {
			log.Fatal(err)
		}

		syncpath, err := dm.GetSyncpathForGame(gamename)
		if err != nil {
			log.Fatal(err)
		}

		err = syncFiles(id, syncpath, files)
		if err != nil {
			log.Fatal(err)
		}
	}

	// res, err := srv.Files.List().
	// 	Q(fmt.Sprintf("'%v' in parents", saveFolderId)).
	// 	Fields("nextPageToken, files(id, name)").
	// 	Do()

	// if len(res.Files) == 0 {
	// 	fmt.Println(err)
	// 	fmt.Println("No files in steamsave....")
	// 	// log.Fatalf("Unable to retrieve files: %v", err)
	// 	ff := &drive.File{
	// 		Name: "TestFile",
	// 		// ModifiedTime: time.Now().GoString(),
	// 		Parents: []string{saveFolderId},
	// 	}

	// 	osf, err := os.Open("test_payload.file")
	// 	defer osf.Close()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	_, err = srv.Files.Create(ff).Media(osf).Do()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// } else {
	// 	if len(res.Files) == 0 {
	// 		fmt.Println("No files found.")
	// 	} else {
	// 		for _, i := range res.Files {
	// 			gg, err := srv.Files.Get(i.Id).Fields("modifiedTime").Do()
	// 			if err != nil {
	// 				log.Fatal(err)
	// 			}
	// 			fmt.Println(gg.ModifiedTime)
	// 			fmt.Printf("%s (%s)\n", i.Name, i.Id)
	// 		}
	// 	}
	// }

	// dm := &DriverManager{}
	// dm.Push("game", []string{"foo"})

}
