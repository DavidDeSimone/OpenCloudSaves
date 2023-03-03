package core

type LibraryFolders struct {
	LibraryFolders map[string]LibraryFolder `json:"libraryfolders"`
}

type LibraryFolder struct {
	Path string            `json:"path"`
	Apps map[string]string `json:"apps"`
}

type AppManifest struct {
	AppState struct {
		InstallDir string `json:"installdir"`
	} `json:"AppState"`
}
