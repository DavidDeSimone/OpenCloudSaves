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
	DownloadFile(fileId string, filePath string, fileName string) (CloudFile, error)
	UploadFile(fileId string, filePath string, fileName string) (CloudFile, error)
	CreateFile(parentId string, fileName string, filePath string) (CloudFile, error)
	DeleteFile(fileId string) error
}
