package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type GoogleCloudFile struct {
	Name    string
	Id      string
	ModTime string
}

func (f *GoogleCloudFile) GetName() string {
	return f.Name
}

func (f *GoogleCloudFile) GetId() string {
	return f.Id
}

func (f *GoogleCloudFile) GetModTime() string {
	return f.ModTime
}

type GoogleCloudDriver struct {
	srv *drive.Service
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

func (d *GoogleCloudDriver) InitDriver() error {
	d.srv = makeService()
	return nil
}
func (d *GoogleCloudDriver) ListFiles(parentId string) ([]CloudFile, error) {
	result := []CloudFile{}

	r, err := d.srv.Files.List().
		Q(fmt.Sprintf("'%v' in parents", parentId)).
		Fields("nextPageToken, files(id, name)").
		Do()
	if err != nil {
		return result, err
	}

	for _, i := range r.Files {
		fileref, err := d.srv.Files.Get(i.Id).Fields("modifiedTime").Do()
		if err != nil {
			return result, err
		}

		cloudFile := &GoogleCloudFile{
			Name:    i.Name,
			Id:      i.Id,
			ModTime: fileref.ModifiedTime,
		}

		result = append(result, cloudFile)
	}

	return result, nil
}
func (d *GoogleCloudDriver) CreateDir(name string, parentId string) (CloudFile, error) {
	f := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
	}

	if parentId != "" {
		f.Parents = []string{parentId}
	}

	result, err := d.srv.Files.Create(f).Do()
	if err != nil {
		return nil, err
	}

	return &GoogleCloudFile{
		Name:    name,
		Id:      result.Id,
		ModTime: result.ModifiedTime,
	}, nil
}
func (d *GoogleCloudDriver) DownloadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	fileref, err := d.srv.Files.Get(fileId).Fields("modifiedTime").Do()
	if err != nil {
		return nil, err
	}

	res, err := d.srv.Files.Get(fileId).Download()
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	osf, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	io.Copy(osf, res.Body)
	osf.Close()
	modtime, err := time.Parse(time.RFC3339, fileref.ModifiedTime)
	if err != nil {
		return nil, err
	}

	err = os.Chtimes(filePath, modtime, modtime)
	if err != nil {
		return nil, err
	}

	return &GoogleCloudFile{
		Name:    fileName,
		Id:      fileId,
		ModTime: fileref.ModifiedTime,
	}, nil
}
func (d *GoogleCloudDriver) UploadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	osf, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer osf.Close()
	stat, err := osf.Stat()
	if err != nil {
		return nil, err
	}

	modifiedAtTime := stat.ModTime().Format(time.RFC3339)
	ff := &drive.File{
		ModifiedTime: modifiedAtTime,
	}

	res, err := d.srv.Files.Update(fileId, ff).Media(osf).Do()
	if err != nil {
		return nil, err
	}
	return &GoogleCloudFile{
		Name:    fileName,
		Id:      res.Id,
		ModTime: modifiedAtTime,
	}, nil
}
func (d *GoogleCloudDriver) CreateFile(parentId string, fileName string, filePath string) (CloudFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	modtime := stat.ModTime().Format(time.RFC3339)
	saveUpload := &drive.File{
		Name:         fileName,
		ModifiedTime: modtime,
		Parents:      []string{parentId},
	}

	result, err := d.srv.Files.Create(saveUpload).Media(file).Do()
	if err != nil {
		return nil, err
	}

	return &GoogleCloudFile{
		Name:    result.Name,
		Id:      result.Id,
		ModTime: modtime,
	}, nil
}

func (d *GoogleCloudDriver) GetMetaData(parentId string, fileName string) (*GameMetadata, error) {
	r, err := d.srv.Files.List().
		Q(fmt.Sprintf("'%v' in parents", parentId)).
		Fields("nextPageToken, files(id, name)").
		Do()

	if err != nil {
		return nil, err
	}

	var metadata *GameMetadata = nil
	for _, file := range r.Files {
		if file.Name == fileName {
			res, err := d.srv.Files.Get(file.Id).Download()
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}

			metadata = &GameMetadata{}
			err = json.Unmarshal(bytes, metadata)
			if err != nil {
				return nil, err
			}
			metadata.fileId = file.Id
			break
		}
	}

	return metadata, nil
}

func (d *GoogleCloudDriver) UpdateMetaData(parentId string, fileName string, filePath string, metaData *GameMetadata) error {
	mf, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer mf.Close()

	metaUpload := &drive.File{
		Name: fileName,
	}

	if metaData.fileId == "" {
		metaUpload.Parents = []string{parentId}
		_, err = d.srv.Files.Create(metaUpload).Media(mf).Do()
		if err != nil {
			return err
		}
	} else {
		_, err = d.srv.Files.Update(metaData.fileId, metaUpload).Media(mf).Do()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *GoogleCloudDriver) DeleteFile(fileId string) error {
	return d.srv.Files.Delete(fileId).Do()
}

func (d *GoogleCloudDriver) IsFileInSync(fileName string, filePath string, fileId string, metaData *GameMetadata) (int, error) {
	meta, ok := metaData.Files[fileName]
	modifiedTime := ""
	remoteFileHash := ""
	if !ok {
		fmt.Println("Remote file not found: " + fileName)
		result, err := d.srv.Files.Get(fileId).Fields("modifiedTime").Do()
		if err != nil {
			return NotFound, nil
		}

		res, err := d.srv.Files.Get(fileId).Download()
		if err != nil {
			return 0, err
		}

		defer res.Body.Close()

		h := sha256.New()
		if _, err := io.Copy(h, res.Body); err != nil {
			log.Fatal(err)
		}

		if err != nil {
			return 0, err
		}

		remoteFileHash = hex.EncodeToString(h.Sum(nil))
		modifiedTime = result.ModifiedTime
	} else {
		modifiedTime = meta.LastModified
		remoteFileHash = meta.Sha256
	}
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	localfile, err := f.Stat()
	if err != nil {
		return 0, err
	}

	localModtime := localfile.ModTime()
	remoteModtime, err := time.Parse(time.RFC3339, modifiedTime)
	if err != nil {
		return 0, err
	}

	localFileHash, err := getFileHash(filePath)
	if err != nil {
		return 0, err
	}

	if localModtime.Equal(remoteModtime) || remoteFileHash == localFileHash {
		return InSync, nil
	} else if remoteModtime.After(localModtime) {
		return RemoteNewer, nil
	} else {
		return LocalNewer, nil
	}
}
