package main

type CloudFile interface {
	GetName() string
	GetId() string
	GetModTime() string
}

type CloudDriver interface {
	InitDriver() error
	ListFiles(parentId string) ([]CloudFile, error)
	CreateDir(name string, parentId string) (CloudFile, error)
	DownloadFile(fileId string, fileName string) error
	UploadFile(fileId string, filePath string, fileName string) (CloudFile, error)
	CreateFile(fileName string, filePath string, parentId string) (CloudFile, error)
}
