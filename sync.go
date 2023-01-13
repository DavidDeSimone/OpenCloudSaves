package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var conflictMutex sync.Mutex
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

func GetLocalMetadata(filePath string, localfs LocalFs) (*GameMetadata, error) {
	localMetafile, err := localfs.ReadFile(filePath)
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

func compress(src string, buf io.Writer) error {
	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	// is file a folder?
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		// get header
		header, err := tar.FileInfoHeader(fi, src)
		if err != nil {
			return err
		}
		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// get content
		data, err := os.Open(src)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, data); err != nil {
			return err
		}
	} else if mode.IsDir() { // folder

		// walk through every file in the folder
		filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
			// generate tar header
			header, err := tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			// must provide real name
			// (see https://golang.org/src/archive/tar/common.go?#L626)
			header.Name = filepath.ToSlash(file)

			// write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				if _, err := io.Copy(tw, data); err != nil {
					return err
				}
			}
			return nil
		})
	} else {
		return fmt.Errorf("error: file type not supported")
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

func decompress(src io.Reader, dst string) error {
	// ungzip
	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	// untar
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := header.Name

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", target)
		}

		// add dst + re-format slashes according to system
		target = filepath.Join(dst, header.Name)
		// if no join is needed, replace with ToSlash:
		// target = filepath.ToSlash(header.Name)

		// check the type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it (with 0755 permission)
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it (with same permission)
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			if _, err := io.Copy(fileToWrite, tr); err != nil {
				return err
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	//
	return nil
}

func SyncFiles(srv CloudDriver, parentId string, syncDataPath Datapath, files map[string]SyncFile, dryrun bool, localfs LocalFs, logs chan Message, cancel chan Cancellation) error {
	syncPath := syncDataPath.Path
	LogMessage(logs, "Syncing Files for %v", syncPath)

	downloadAuthorized := syncDataPath.NetAuth&CloudOperationDownload != 0
	uploadAuthorized := syncDataPath.NetAuth&CloudOperationUpload != 0
	deleteAuthoirzed := syncDataPath.NetAuth&CloudOperationDelete != 0

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

	dirList := []string{}
	for k, v := range files {
		if v.IsDir {
			pid, err := CreateRemoteDirIfNotExists(srv, parentId, k)
			if err != nil {
				return err
			}
			separator := string(os.PathSeparator)
			parentPath := Datapath{
				Path:    syncPath + separator + k + separator,
				Parent:  k,
				NetAuth: syncDataPath.NetAuth,
			}
			var fileMap map[string]SyncFile = make(map[string]SyncFile)
			cancelErr = checkIfShouldCancel(cancel)
			if cancelErr != nil {
				return cancelErr
			}

			filesInDir, err := localfs.ReadDir(syncPath + separator + k + separator)
			if err != nil {
				return err
			}

			for _, file := range filesInDir {
				isDir := false
				if file.IsDir() {
					isDir = true
				}

				fileMap[file.Name()] = SyncFile{
					Name:  parentPath.Path + file.Name(),
					IsDir: isDir,
				}
			}

			cancelErr = checkIfShouldCancel(cancel)
			if cancelErr != nil {
				return cancelErr
			}

			// @TODO why not just make this simpler and tarball the whole thing...?
			// var buf bytes.Buffer
			// err = compress("./folderToCompress", &buf)
			// if err != nil {

			// }

			// // write the .tar.gzip
			// fileToWrite, err := os.OpenFile("./compress.tar.gzip", os.O_CREATE|os.O_RDWR, os.FileMode(600))
			// if err != nil {
			// 	panic(err)
			// }
			// if _, err := io.Copy(fileToWrite, &buf); err != nil {
			// 	panic(err)
			// }

			// // untar write
			// if err := untar(&buf, "./uncompressHere/"); err != nil {
			// 	// probably delete uncompressHere?
			// }

			err = SyncFiles(srv, pid, parentPath, fileMap, dryrun, localfs, logs, cancel)
			if err != nil {
				return err
			}

			dirList = append(dirList, k)
		}
	}

	cancelErr = checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	for _, d := range dirList {
		delete(files, d)
	}

	clientuuid, err := GetClientUUID()
	LogMessage(logs, "Identified Client UUID (%v)", clientuuid)
	if err != nil {
		return err
	}
	// 1. Query current files on cloud:
	r, err := srv.ListFiles(parentId)
	if err != nil {
		return err
	}

	cancelErr = checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	metadata, err := srv.GetMetaData(parentId, STEAM_METAFILE)
	if err != nil {
		return err
	}
	if metadata == nil {
		LogMessage(logs, "Did not find remote metafile, initalizing... %v", parentId)
		metadata = &GameMetadata{
			Version: CURRENT_META_VERSION,
			Gameid:  parentId,
			Files:   make(map[string]FileMetadata),
		}
	}

	localMetadata, err := GetLocalMetadata(syncPath+STEAM_METAFILE, localfs)
	if err != nil {
		return err
	}

	LogMessage(logs, "-------- STAGE 1 -----------")
	LogMessage(logs, "Examining files present on cloud but deleted locally")
	var deletedFiles map[string]bool = make(map[string]bool)

	if !deleteAuthoirzed {
		LogMessage(logs, "Skipping deletions for %v", syncDataPath.Path)
	}

	cancelErr = checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	// 1. Handle the case for deleting save data
	if localMetadata != nil && deleteAuthoirzed {
		// If a file is in our local metafile, but not present locally, delete on cloud.
		for k := range localMetadata.Files {

			cancelErr = checkIfShouldCancel(cancel)
			if cancelErr != nil {
				return cancelErr
			}

			if _, err := localfs.Stat(syncPath + k); errors.Is(err, os.ErrNotExist) {
				for _, f := range r {
					if f.GetName() == k {
						if dryrun {
							fmt.Printf("Dry Run - Would Delete %v on cloud (not really deleting\n", f.GetName())
						} else {
							if localMetadata.Files[k].Sha256 != metadata.Files[k].Sha256 {
								// CONFLICT - the file that we plan on deleting is NOT the same as on the server
								// We should surface to the user if we want to delete this.
								LogMessage(logs, ">>>>> Deleted File Modified on the Cloud <<<<<<<<<<")
								LogMessage(logs, "There is a file on the cloud that you have deleted locally - %v. Local hash %v, remote hash %v", k, localMetadata.Files[k].Sha256, metadata.Files[k].Sha256)
								LogMessage(logs, "Press d to (d)elete the file on the cloud. Press k to (k)eep the file on the cloud. Press s to (s)kip")

								conflictMutex.Lock()
								input := ""
								defer conflictMutex.Unlock()

								for {
									fmt.Scan(&input)

									if input == "d" {
										// Delete the remote file
										err = srv.DeleteFile(f.GetId())
										if err != nil {
											return err
										}
										delete(metadata.Files, k)
										deletedFiles[k] = true
										break
									} else if input == "k" {
										// Remove it from the local metadata, keep on trucking.
										delete(metadata.Files, k)
										break
									} else if input == "s" {
										break
									} else {
										LogMessage(logs, "Please enter one of the following options: d, k, or s...")
									}
								}

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

		cancelErr = checkIfShouldCancel(cancel)
		if cancelErr != nil {
			return cancelErr
		}

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
			if !downloadAuthorized {
				LogMessage(logs, "Skipping download for %v", file.GetName())
				continue
			}

			// 2a. Not present on local file system, download...
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
				if !downloadAuthorized {
					LogMessage(logs, "Skipping download for %v", file.GetName())
					continue
				}

				inputChannel <- SyncRequest{
					Operation: Download,
					FileId:    file.GetId(),
					Path:      fullpath,
					Name:      file.GetName(),
					Dryrun:    dryrun,
				}

				pendingUploadDownload += 1
			} else {
				if !uploadAuthorized {
					LogMessage(logs, "Skipping Upload for %v", file.GetName())
					continue
				}

				inputChannel <- SyncRequest{
					Operation: Upload,
					FileId:    file.GetId(),
					Path:      fullpath,
					Name:      file.GetName(),
					Dryrun:    dryrun,
				}

				pendingUploadDownload += 1
			}
		}
	}

	totalPendingOperations := pendingUploadDownload
	lastPercentage := 0
	for pendingUploadDownload > 0 {
		cancelErr = checkIfShouldCancel(cancel)
		if cancelErr != nil {
			return cancelErr
		}

		response := <-outputChannel
		if response.Err != nil {
			return response.Err
		}

		newModifiedTime := response.Result
		fullpath := response.Path
		fileName := response.Name

		newFileHash, err := localfs.GetFileHash(fullpath)
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

		percentage := int(1.0 - (float64(pendingUploadDownload)/float64(totalPendingOperations))*100)
		if percentage > lastPercentage {
			LogMessage(logs, "Percentage Complete: %v%%", percentage)
			lastPercentage = percentage
		}
	}

	cancelErr = checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	LogMessage(logs, "-------- STAGE 3 -----------")
	LogMessage(logs, "Download new files from remote")
	// 3. Check for local files not present on the cloud
	numCreations := 0
	for k, v := range files {
		cancelErr = checkIfShouldCancel(cancel)
		if cancelErr != nil {
			return cancelErr
		}

		if !uploadAuthorized {
			LogMessage(logs, "Skipping Initial File Uploads")
			break
		}

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
		}
	}

	LogMessage(logs, "Queue upload for %v files", numCreations)
	totalCreations := numCreations
	lastPercentage = 0
	for numCreations > 0 {
		cancelErr = checkIfShouldCancel(cancel)
		if cancelErr != nil {
			return cancelErr
		}

		results := <-outputChannel
		if results.Err != nil {
			return results.Err
		}

		filehash := "foo"
		// filehash, err := localfs.GetFileHash(results.Path)
		// if err != nil {
		// 	return err
		// }

		metadata.Files[results.Name] = FileMetadata{
			Sha256:         filehash,
			LastModified:   results.Result,
			Lastclientuuid: clientuuid,
			FileId:         results.FileId,
		}

		numCreations -= 1
		percentage := int((1.0 - (float64(numCreations) / float64(totalCreations))) * 100)
		if percentage > lastPercentage {
			LogMessage(logs, "Percentage Complete: %v%%", percentage)
			lastPercentage = percentage
		}

	}

	cancelErr = checkIfShouldCancel(cancel)
	if cancelErr != nil {
		return cancelErr
	}

	LogMessage(logs, "Data Upload/Download success - updating metadata...")
	metadata.Version = CURRENT_META_VERSION
	metadata.ParentId = parentId
	b, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	err = localfs.WriteFile(syncPath+STEAM_METAFILE, b, os.ModePerm)

	if err != nil {
		return nil
	}

	err = srv.UpdateMetaData(parentId, STEAM_METAFILE, syncPath+STEAM_METAFILE, metadata)
	if err != nil {
		return err
	}

	return nil
}
