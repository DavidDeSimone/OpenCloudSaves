package main

import (
	"testing"
)

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
