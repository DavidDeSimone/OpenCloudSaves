package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CloudRecord struct {
	Files []string `json:"files"`
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

func zipSource(source, target string, filter func(string) bool) ([]string, error) {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	fileList := []string{}
	// 2. Go through all the files of the source
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		referenceName := strings.Replace(path, source, "", 1)
		fmt.Println("Examining " + referenceName)
		if filter(referenceName) {
			fmt.Println("Ignoring File " + referenceName)
			return nil
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
			fileList = append(fileList, referenceName)
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		fileList = append(fileList, referenceName)
		_, err = io.Copy(headerWriter, f)
		return err
	})

	return fileList, err
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

	// 6. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	// 7. Create a destination file for unzipped content
	destinationStat, err := os.Stat(filePath)
	if err != nil || f.Modified.After(destinationStat.ModTime()) {
		destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer destinationFile.Close()

		if _, err := io.Copy(destinationFile, zippedFile); err != nil {
			return err
		}
	}

	return nil
}

func SyncFiles(srv CloudDriver, parentId string, syncDataPath Datapath, channels *ChannelProvider) error {
	logs := channels.logs
	cancel := channels.cancel
	syncPath := syncDataPath.Path
	LogMessage(logs, "Syncing Files for %v", syncPath)

	downloadAuthorized := syncDataPath.NetAuth&CloudOperationDownload != 0
	uploadAuthorized := syncDataPath.NetAuth&CloudOperationUpload != 0
	deleteAuthoirzed := syncDataPath.NetAuth&CloudOperationDelete != 0

	inputChannel := channels.input
	outputChannel := channels.output

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	record := &CloudRecord{}
	metaFileName := cacheDir + string(os.PathSeparator) + parentId + ".json"
	if _, err := os.Stat(metaFileName); !errors.Is(err, os.ErrNotExist) {
		buf, err := os.ReadFile(metaFileName)
		if err != nil {
			return err
		}

		err = json.Unmarshal(buf, record)
		if err != nil {
			return err
		}
	}

	cancelErr := checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	deleteList := []string{}
	for _, file := range record.Files {
		if _, err := os.Stat(syncPath + file); errors.Is(err, os.ErrNotExist) {
			deleteList = append(deleteList, syncPath+file)
		}
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

	if deleteAuthoirzed {
		for _, file := range deleteList {
			// Since deleting a file may result in a
			// recursive delete, this can fail based on delete
			// order. Think deleting dir and dir/file.txt
			// where you delete dir first.
			err = os.RemoveAll(file)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	os.Remove(filePath)
	fileList, err := zipSource(syncDataPath.Path, filePath,
		func(s string) bool {
			for _, ignore := range syncDataPath.Ignore {
				if s == ignore {
					return true
				}
			}

			if len(syncDataPath.Exts) == 0 {
				return false
			}

			anyExtMatches := false
			fileExt := filepath.Ext(s)
			for _, ext := range syncDataPath.Exts {
				if ext == fileExt {
					anyExtMatches = true
					break
				}
			}

			return !anyExtMatches

		})
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

	record = &CloudRecord{
		Files: fileList,
	}
	buf, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = os.WriteFile(metaFileName, buf, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
