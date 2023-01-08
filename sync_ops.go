package main

const (
	Create = iota
	Download
	Upload
)

type SyncRequest struct {
	Operation int
	Name      string
	Path      string
	ParentId  string
	FileId    string
	Dryrun    bool
}

type SyncResponse struct {
	Operation int
	Result    string
	Name      string
	Path      string
	FileId    string
	Err       error
}

func SyncOp(srv CloudDriver, input chan SyncRequest, output chan SyncResponse) {
	for {
		request := <-input
		switch request.Operation {
		case Create:
			CreateOperation(srv, request, output)
		case Download:
			DownloadOperation(srv, request, output)
		case Upload:
			UploadOperation(srv, request, output)
		}
	}
}

func CreateOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.CreateFile(request.ParentId, request.Name, request.Path)
	resultModtime := ""
	resultFileId := ""
	if result != nil {
		resultModtime = result.GetModTime()
		resultFileId = result.GetId()
	}

	output <- SyncResponse{
		Operation: Create,
		Name:      request.Name,
		Path:      request.Path,
		Result:    resultModtime,
		FileId:    resultFileId,
		Err:       err,
	}
}

func DownloadOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.DownloadFile(request.FileId, request.Path, request.Name) //downloadFile(srv, request.FileId, request.Name, request.Dryrun)
	resultModtime := ""
	resultFileId := ""
	if result != nil {
		resultModtime = result.GetModTime()
		resultFileId = result.GetId()
	}

	output <- SyncResponse{
		Operation: Download,
		Name:      request.Name,
		Path:      request.Path,
		FileId:    resultFileId,
		Result:    resultModtime,
		Err:       err,
	}
}

func UploadOperation(srv CloudDriver, request SyncRequest, output chan SyncResponse) {
	result, err := srv.UploadFile(request.FileId, request.Path, request.Name) //uploadFile(srv, request.FileId, request.Path, request.Dryrun)
	resultModtime := ""
	resultFileId := ""
	if result != nil {
		resultModtime = result.GetModTime()
		resultFileId = result.GetId()
	}

	output <- SyncResponse{
		Operation: Download,
		Result:    resultModtime,
		Name:      request.Name,
		Path:      request.Path,
		FileId:    resultFileId,
		Err:       err,
	}
}
