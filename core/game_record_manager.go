package core

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-yaml/yaml"
)

type FileConstraint struct {
	Os    Os
	Store Store
}

type LaunchConstraint struct {
	Bit   Bit   // 32 or 64
	Os    Os    // windows, linux, darwin
	Store Store // steam, gog, etc.
}

type RegistryConstraint struct {
	Store Store // steam, gog, etc.
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
	GetManifestLocation() string
	GetGameRecordByKey(key string) (*GameRecord, error)
	SetGameRecordByKey(key string, grm *GameRecord) error
	SetGameRecordManifest(map[string]*GameRecord) error
	RefreshManifestFromDisk() error
	CommitCustomGameRecords() error
	VisitGameRecords(func(key string, grm *GameRecord) error) error
	VisitGameRecord(key string, cb func(grm *GameRecord) error) error
}

type GameRecordManagerImpl struct {
	manifestLocation string
	gameRecords      map[string]*GameRecord
	mu               sync.Mutex
}

var gameRecordManagerInstance GameRecordManager

func GetGameRecordManager() GameRecordManager {
	if gameRecordManagerInstance == nil {
		gameRecordManagerInstance = NewGameRecordManager()
		InitializeGameRecordManager(gameRecordManagerInstance)
	}

	return gameRecordManagerInstance
}

// parse a game record manifest from a []byte
func ParseGameRecordManifest(content []byte) (map[string]*GameRecord, error) {
	records := make(map[string]*GameRecord)
	err := yaml.Unmarshal(content, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// create a game record fetcher,
// fetch the manifest,
// parse the manifest,
// and store the game records in the game record manager
func InitializeGameRecordManager(grm GameRecordManager) {
	fetcher := NewGameRecordManifestFetcher()
	content, _, err := fetcher.Fetch()
	var gameRecords map[string]*GameRecord
	if err != nil {
		if errors.Is(err, ErrManifestNotModified) {
			rawBytes, err := os.ReadFile(grm.GetManifestLocation())
			if err != nil {
				panic(err)
			}

			var b bytes.Buffer
			decoder := gob.NewDecoder(&b)
			b.Write(rawBytes)
			err = decoder.Decode(&gameRecords)
			if err != nil {
				panic(err)
			}
			content = b.Bytes()

			if err != nil {
				// @TODO delete tag file, try fetching again
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		gameRecords, err = ParseGameRecordManifest(content)
		if err != nil {
			panic(err)
		}
		go func() {
			var b bytes.Buffer
			encoder := gob.NewEncoder(&b)
			err = encoder.Encode(gameRecords)
			if err != nil {
				panic(err)
			}

			err = os.WriteFile(grm.GetManifestLocation(), b.Bytes(), 0644)
			if err != nil {
				panic(err)
			}
		}()
	}

	// store the game records in the game record manager
	err = grm.SetGameRecordManifest(gameRecords)
	if err != nil {
		panic(err)
	}
}

func NewGameRecordManager() GameRecordManager {
	cacheDir, _ := os.UserConfigDir()

	return &GameRecordManagerImpl{
		gameRecords:      make(map[string]*GameRecord),
		manifestLocation: filepath.Join(cacheDir, APP_NAME, "manifest.blob"),
	}
}

func (grm *GameRecordManagerImpl) VisitGameRecords(fn func(key string, grm *GameRecord) error) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	fmt.Println("Conunt = " + fmt.Sprint(len(grm.gameRecords)))
	for key, gr := range grm.gameRecords {
		err := fn(key, gr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (grm *GameRecordManagerImpl) VisitGameRecord(key string, fn func(grm *GameRecord) error) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	gr, exists := grm.gameRecords[key]
	if !exists {
		return errors.New("game record not found")
	}

	return fn(gr)
}

func (grm *GameRecordManagerImpl) SetGameRecordManifest(records map[string]*GameRecord) error {
	grm.mu.Lock()
	defer grm.mu.Unlock()

	grm.gameRecords = records
	return nil
}

func (grm *GameRecordManagerImpl) SetManifestLocation(filepath string) {
	grm.manifestLocation = filepath
}

func (grm *GameRecordManagerImpl) GetManifestLocation() string {
	return grm.manifestLocation
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
