package core

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
)

type SyncRequest struct {
	ctx      context.Context
	path     string
	response chan error
}

type UserSettingsManager struct {
	syncRequests  chan SyncRequest
	asyncRequests chan SyncRequest
	quit          chan struct{}
	wg            sync.WaitGroup
	cm            *CloudManager
}

var userSettingsManager *UserSettingsManager

func GetUserSettingsManager() *UserSettingsManager {
	if userSettingsManager == nil {
		userSettingsManager = NewUserSettingsManager(MakeCloudManager())
	}

	return userSettingsManager
}

func NewUserSettingsManager(cm *CloudManager) *UserSettingsManager {
	usm := &UserSettingsManager{
		syncRequests:  make(chan SyncRequest),
		asyncRequests: make(chan SyncRequest),
		quit:          make(chan struct{}),
		cm:            cm,
	}

	usm.wg.Add(1)
	go usm.run()

	return usm
}

func (usm *UserSettingsManager) run() {
	defer usm.wg.Done()

	for {
		select {
		case req := <-usm.syncRequests:
			err := usm.syncUserSettings(req.ctx, req.path)
			req.response <- err
		case asyncReq := <-usm.asyncRequests:
			go func() {
				err := usm.syncUserSettings(asyncReq.ctx, asyncReq.path)
				if asyncReq.response != nil {
					asyncReq.response <- err
				}
			}()
		case <-usm.quit:
			return
		}
	}
}

func (usm *UserSettingsManager) syncUserSettings(ctx context.Context, userOverridePath string) error {
	if usm.cm == nil {
		usm.cm = MakeCloudManager()
	}

	userOverride := userOverridePath
	storage := GetCurrentStorageProvider()
	if storage == nil {
		return errors.New("no cloud storage set")
	}

	path := filepath.Dir(userOverride)
	ops := GetDefaultCloudOptions()
	ops.Include = "*.json"
	_, err := usm.cm.PerformSyncOperation(ctx, storage, ops, path, ToplevelCloudFolder+"user_settings/")
	return err
}

func (usm *UserSettingsManager) RequestSync(ctx context.Context, path string) error {
	if path == "" {
		path = GetDefaultUserOverridePath()
	}

	response := make(chan error)
	usm.syncRequests <- SyncRequest{
		ctx:      ctx,
		path:     path,
		response: response,
	}
	return <-response
}

func (usm *UserSettingsManager) RequestSyncNonBlocking(ctx context.Context, path string, res chan error) {
	if path == "" {
		path = GetDefaultUserOverridePath()
	}

	usm.asyncRequests <- SyncRequest{
		ctx:      ctx,
		path:     path,
		response: res,
	}
}

func (usm *UserSettingsManager) Stop() {
	close(usm.quit)
	usm.wg.Wait()
}
