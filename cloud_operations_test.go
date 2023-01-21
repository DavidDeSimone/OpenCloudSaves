package main

import (
	"fmt"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
	// Override global cloud service
	service = &MockCloudDriver{}
	service.InitDriver()
	dm := MakeGameDefManager("tests/tests_useroverrides.json")
	ops := &Options{
		Gamenames: []string{t.Name()},
	}

	channels := &ChannelProvider{
		logs:     make(chan Message, 100),
		cancel:   make(chan Cancellation, 1),
		input:    make(chan SyncRequest, 10),
		output:   make(chan SyncResponse, 10),
		progress: make(chan ProgressEvent, 15),
	}

	go CliMain(ops, dm, channels, SyncOp)

	for {

		select {
		case result := <-channels.logs:
			if result.Finished {
				return
			}

			if result.Err != nil {
				fmt.Println(result.Err)
			} else {
				fmt.Println(result.Message)
			}
		case <-time.After(20 * time.Second):
			t.Error("timeout")
		}
	}
}
