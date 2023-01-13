package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var clientUUID string

func GetClientUUID() (string, error) {
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

func checkIfShouldCancel(cancelChannel chan Cancellation) error {
	select {
	case msg := <-cancelChannel:
		if msg.ShouldCancel {
			return errors.New("request Cancelled")
		}
	default:
		return nil
	}

	return nil
}

func CreateRemoteDirIfNotExists(srv CloudDriver, parentId string, name string) (string, error) {
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

func ValidateAndCreateParentFolder(srv CloudDriver) (string, error) {
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

func zipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		fmt.Println("Copying " + path)
		_, err = io.Copy(headerWriter, f)
		return err
	})
}

func unzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	fmt.Println(destination)
	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	// if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
	//     return fmt.Errorf("invalid file path: %s", filePath)
	// }

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	destinationStat, err := os.Stat(filePath)
	if err != nil || f.Modified.After(destinationStat.ModTime()) {
		destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer destinationFile.Close()

		fmt.Println("Overwriting file " + destinationFile.Name())
		if _, err := io.Copy(destinationFile, zippedFile); err != nil {
			return err
		}
	} else {
		fmt.Println("Not modifying " + filePath)
	}

	return nil
}

// @TODO a better handling of delete
func SyncFiles(srv CloudDriver, parentId string, syncDataPath Datapath, channels *ChannelProvider) error {
	logs := channels.logs
	cancel := channels.cancel
	syncPath := syncDataPath.Path
	LogMessage(logs, "Syncing Files for %v", syncPath)

	downloadAuthorized := syncDataPath.NetAuth&CloudOperationDownload != 0
	uploadAuthorized := syncDataPath.NetAuth&CloudOperationUpload != 0
	// deleteAuthoirzed := syncDataPath.NetAuth&CloudOperationDelete != 0

	// Test if folder exists, and if it does, what it contains
	// Update folder with data if file names match and files are newer
	inputChannel := make(chan SyncRequest, 1000)
	outputChannel := make(chan SyncResponse, 1000)
	for i := 0; i < WORKER_POOL_SIZE; i++ {
		go SyncOp(srv, inputChannel, outputChannel)
	}

	cancelErr := checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	zipExists := false

	fileName := fmt.Sprintf("%v.zip", syncDataPath.Parent)
	filePath := fmt.Sprintf("%v%v", os.TempDir(), fileName)
	cloudFiles, err := srv.ListFiles(parentId)
	if err != nil {
		return err
	}

	var exisitingCloudFile CloudFile = nil
	for _, cloudFile := range cloudFiles {
		if cloudFile.GetName() == fileName {
			zipExists = true
			exisitingCloudFile = cloudFile
			break
		}
	}

	if exisitingCloudFile != nil && downloadAuthorized {
		inputChannel <- SyncRequest{
			Operation: Download,
			Name:      fileName,
			Path:      filePath,
			ParentId:  parentId,
			FileId:    exisitingCloudFile.GetId(),
		}

		downloadResult := <-outputChannel
		if downloadResult.Err != nil {
			return downloadResult.Err
		}

		err = unzipSource(filePath, syncDataPath.Path)
		if err != nil {
			return err
		}
	}

	os.Remove(filePath)
	err = zipSource(syncDataPath.Path, filePath)
	if err != nil {
		return err
	}

	operation := Create
	if zipExists {
		operation = Upload
	}

	if uploadAuthorized {
		syncRequest := SyncRequest{
			Operation: operation,
			Name:      fileName,
			Path:      filePath,
			ParentId:  parentId,
		}

		if exisitingCloudFile != nil && operation == Upload {
			syncRequest.FileId = exisitingCloudFile.GetId()
		}

		inputChannel <- syncRequest

		result := <-outputChannel
		if result.Err != nil {
			return result.Err
		}
	}

	return nil
}
