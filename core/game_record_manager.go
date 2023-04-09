package core

import (
	"errors"
	"os"
	"sync"

	"github.com/go-yaml/yaml"
	_ "github.com/stretchr/testify/assert"
)

type GameRecordManifestFetcher interface {
	Fetch()
}

type GameRecordManifestFetcherImpl struct {
}

func (fetcher *GameRecordManifestFetcherImpl) Fetch() {
	// @TODO
}

type FileConstraint struct {
	Os    Os
	Store Store
}

type LaunchConstraint struct {
	Bit   Bit
	Os    Os
	Store Store
}

type RegistryConstraint struct {
	Store Store
}

type Bit int
type Os string
type Store string
type Tag string

type FileProperties struct {
	Tags []Tag
	When []FileConstraint
}

type LaunchProperties struct {
	Arguments  string
	WorkingDir string
	When       []LaunchConstraint
}

type RegistryProperties struct {
	Tags []Tag
	When []RegistryConstraint
}

type SteamProperties struct {
	Id int
}

type GogProperties struct {
	Id int
}

type GameRecord struct {
	Files      map[string]FileProperties
	InstallDir map[string]interface{}
	Launch     map[string][]LaunchProperties
	Registry   map[string]RegistryProperties
	Steam      SteamProperties
	Gog        GogProperties
}

type GameRecordManager interface {
	SetManifestLocation(filepath string)
	GetGameRecordByKey(key string) (*GameRecord, error)
	SetGameRecordByKey(key string, grm *GameRecord) error
	RefreshManifestFromDisk() error
	CommitCustomGameRecords() error
}

type GameRecordManagerImpl struct {
	manifestLocation string
	gameRecords      map[string]*GameRecord
	mu               sync.Mutex
}

func NewGameRecordManager() GameRecordManager {
	return &GameRecordManagerImpl{
		gameRecords: make(map[string]*GameRecord),
	}
}

func (grm *GameRecordManagerImpl) SetManifestLocation(filepath string) {
	grm.manifestLocation = filepath
}

func (grm *GameRecordManagerImpl) GetGameRecordByKey(key string) (*GameRecord, error) {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	record, exists := grm.gameRecords[key]
	if !exists {
		return nil, errors.New("game record not found")
	}

	return record, nil
}

func (grm *GameRecordManagerImpl) SetGameRecordByKey(key string, gr *GameRecord) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	grm.gameRecords[key] = gr
	return nil
}

func (grm *GameRecordManagerImpl) RefreshManifestFromDisk() error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	data, err := os.ReadFile(grm.manifestLocation)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &grm.gameRecords)
	if err != nil {
		return err
	}

	return nil
}

func (grm *GameRecordManagerImpl) CommitCustomGameRecords() error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	data, err := yaml.Marshal(grm.gameRecords)
	if err != nil {
		return err
	}

	err = os.WriteFile(grm.manifestLocation, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
