package main

type CloudFile interface {
	GetName() string
	GetId() string
	GetModTime() string
}

const (
	NotFound    = -1
	InSync      = 0
	LocalNewer  = 1
	RemoteNewer = 2
)

type CloudDriver interface {
	InitDriver(metaData *GameMetadata) error
	ListFiles(parentId string) ([]CloudFile, error)
	CreateDir(name string, parentId string) (CloudFile, error)
	DownloadFile(fileId string, fileName string) error
	UploadFile(fileId string, filePath string, fileName string) (CloudFile, error)
	CreateFile(fileName string, filePath string, parentId string) (CloudFile, error)
	IsFileInSync(fileName string, filePath string, fileId string) (int, error)
}
