package main

import (
	"fmt"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/container"
	"fyne.io/fyne/widget"
)

func GuiMain(ops *Options, dm *GameDefManager) {
	a := app.New()
	w := a.NewWindow("Steam Custom Cloud Uploads")
	w.Resize(fyne.NewSize(500, 500))
	cont := container.NewVBox(widget.NewLabel("Steam Custom Cloud Uploads"))

	syncMap := make(map[string]bool)
	for k, v := range dm.GetGameDefMap() {
		key := k
		cont.Add(widget.NewCheck(v.DisplayName, func(b bool) {
			syncMap[key] = b
		}))
	}

	cont.Add(widget.NewButton("Sync", func() {
		ops.Sync = []bool{true}
		ops.Gamenames = []string{}
		for k, v := range syncMap {
			if v {
				ops.Gamenames = append(ops.Gamenames, k)
			}
		}

		fmt.Println(ops.Gamenames)
		CliMain(ops, dm)
	}))

	scroll := container.NewVScroll(cont)
	w.SetContent(scroll)

	w.ShowAndRun()
}
