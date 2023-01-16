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
	DownloadFile(fileId string, filePath string, fileName string, progress func(int64, int64)) (CloudFile, error)
	UploadFile(fileId string, filePath string, fileName string, progress func(int64, int64)) (CloudFile, error)
	CreateFile(parentId string, fileName string, filePath string, progress func(int64, int64)) (CloudFile, error)
	DeleteFile(fileId string) error
}
