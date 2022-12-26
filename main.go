package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jessevdk/go-flags"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Options struct {
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
}

const SAVE_FOLDER string = "steamsave"

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

func main() {
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope, drive.DriveMetadataReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	// TODO this should be abstracted to be also used with custom drivers
	// right now this is really just my scratchpad for google drive upload APIs
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
			fmt.Printf("%s (%s)\n", i.Name, i.Id)
		}
	}

	if createSaveFolder {
		f := &drive.File{
			Name:     SAVE_FOLDER,
			MimeType: "application/vnd.google-apps.folder",
		}

		fmt.Println("Creating steamsaves folder....")
		x, err := srv.Files.Create(f).Do()
		saveFolderId = x.Id
		if err != nil {
			log.Fatal(err)
		}
	}

	res, err := srv.Files.List().
		Q(fmt.Sprintf("'%v' in parents", saveFolderId)).
		Fields("nextPageToken, files(id, name)").
		Do()

	if len(res.Files) == 0 {
		fmt.Println(err)
		fmt.Println("No files in steamsave....")
		// log.Fatalf("Unable to retrieve files: %v", err)
		ff := &drive.File{
			Name: "TestFile",
			// ModifiedTime: time.Now().GoString(),
			Parents: []string{saveFolderId},
		}

		osf, err := os.Open("test_payload.file")
		defer osf.Close()
		if err != nil {
			log.Fatal(err)
		}

		_, err = srv.Files.Create(ff).Media(osf).Do()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if len(res.Files) == 0 {
			fmt.Println("No files found.")
		} else {
			for _, i := range res.Files {
				fmt.Println(i.ModifiedTime)
				fmt.Printf("%s (%s)\n", i.Name, i.Id)
			}
		}
	}

	ops := &Options{}
	_, err = flags.Parse(ops)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(ops.Verbose))
	// dm := &DriverManager{}
	// dm.Push("game", []string{"foo"})

}
