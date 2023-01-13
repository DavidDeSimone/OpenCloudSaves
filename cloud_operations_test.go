package main

import (
	"testing"
)

// @TODO the way to mock our tests is to inject via the channels
// of sync requests.
func TestBasic(t *testing.T) {

}

// func injectTestGameDef(dm GameDefManager, testDataRoot string, t *testing.T) {
// 	genericDatapath := []*Datapath{
// 		{
// 			Path:    testDataRoot,
// 			Exts:    []string{},
// 			Ignore:  []string{},
// 			Parent:  "saves",
// 			NetAuth: CloudOperationAll,
// 		},
// 	}

// 	dm.GetGameDefMap()[t.Name()] = &GameDef{
// 		DisplayName:          t.Name(),
// 		SteamId:              "0",
// 		SavesCrossCompatible: true,
// 		WinPath:              genericDatapath,
// 		DarwinPath:           genericDatapath,
// 		LinuxPath:            genericDatapath,
// 	}
// }
