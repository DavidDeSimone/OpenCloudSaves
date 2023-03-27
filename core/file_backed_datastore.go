package core

import (
	"context"
	"encoding/json"
	"os"
)

type Datastore[T any] interface {
	Initialize()
	Fetch() (T, error)
	Store(data T)
	Flush() error
}

type CloudBasedDatastore[T any] struct {
	filepath        string
	recv            chan T
	send            chan error
	stop            chan struct{}
	flush           chan struct{}
	cloudManager    CloudManager
	storageProvider Storage
	cloudOptions    CloudOptions
}

func NewCloudBasedDatastore[T any](filepath string, cm CloudManager, sp Storage, co CloudOptions) *CloudBasedDatastore[T] {
	return &CloudBasedDatastore[T]{
		filepath:        filepath,
		cloudManager:    cm,
		storageProvider: sp,
		cloudOptions:    co,
	}
}

func Send[T any](c *CloudBasedDatastore[T]) {
	for {
		select {
		case data := <-c.recv:
			jsonStr, err := json.Marshal(data)
			if err != nil {
				c.send <- err
				continue
			}

			if err = os.WriteFile(c.filepath, jsonStr, os.ModePerm); err != nil {
				c.send <- err
				continue
			}

			cm := c.cloudManager
			storage := c.storageProvider
			ops := c.cloudOptions
			cm.PerformSyncOperation(context.Background(), storage, ops, c.filepath, ToplevelCloudFolder+"user_settings/"+c.filepath)
		case <-c.flush:
			c.send <- nil
		case <-c.stop:
			return
		}
	}
}

func (c *CloudBasedDatastore[T]) Initialize() {
	c.recv = make(chan T)
	c.send = make(chan error)
	c.stop = make(chan struct{})
	c.flush = make(chan struct{})
	go Send(c)
}

func (c *CloudBasedDatastore[T]) Store(data T) {
	c.recv <- data
}

func (c *CloudBasedDatastore[T]) Fetch() (*T, error) {
	cm := c.cloudManager
	storage := c.storageProvider
	ops := c.cloudOptions
	_, err := cm.PerformSyncOperation(context.Background(), storage, ops, ToplevelCloudFolder+"user_settings/"+c.filepath, c.filepath)

	if err != nil {
		return nil, err
	}

	jsonBytes, err := os.ReadFile(c.filepath)
	if err != nil {
		return nil, err
	}

	var data T
	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *CloudBasedDatastore[T]) Flush() error {
	c.flush <- struct{}{}
	err := <-c.send
	return err
}
