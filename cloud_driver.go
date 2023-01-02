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
	InitDriver() error
	ListFiles(parentId string) ([]CloudFile, error)
	CreateDir(name string, parentId string) (CloudFile, error)
	DownloadFile(fileId string, fileName string) error
	UploadFile(fileId string, filePath string, fileName string) (CloudFile, error)
	CreateFile(parentId string, fileName string, filePath string) (CloudFile, error)
	DeleteFile(fileId string) error
	IsFileInSync(fileName string, filePath string, fileId string, metaData *GameMetadata) (int, error)
	GetMetaData(parentId string, fileName string) (*GameMetadata, error)
	UpdateMetaData(parentId string, fileName string, filePath string, metaData *GameMetadata) error
}
