package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

//go:embed icon.jpg
var icon []byte

func makeButtonList(count int) []fyne.CanvasObject {
	var items []fyne.CanvasObject
	for i := 1; i <= count; i++ {
		index := i // capture
		items = append(items, widget.NewButton(fmt.Sprintf("Button %d", index), func() {
			fmt.Println("Tapped", index)
		}))
	}

	return items
}

func makeScrollTab(_ fyne.Window) fyne.CanvasObject {
	hlist := makeButtonList(20)
	vlist := makeButtonList(50)

	horiz := container.NewHScroll(container.NewHBox(hlist...))
	vert := container.NewVScroll(container.NewVBox(vlist...))

	return container.NewAdaptiveGrid(2,
		container.NewBorder(horiz, nil, nil, nil, vert),
		makeScrollBothTab())
}

func makeScrollBothTab() fyne.CanvasObject {
	logo := canvas.NewImageFromResource(theme.FyneLogo())
	logo.SetMinSize(fyne.NewSize(800, 800))

	scroll := container.NewScroll(logo)
	scroll.Resize(fyne.NewSize(400, 400))

	return scroll
}

func makeSplitTab(_ fyne.Window) fyne.CanvasObject {
	left := widget.NewMultiLineEntry()
	left.Wrapping = fyne.TextWrapWord
	left.SetText("Long text is looooooooooooooong")
	right := container.NewVSplit(
		widget.NewLabel("Label"),
		widget.NewButton("Button", func() { fmt.Println("button tapped!") }),
	)
	return container.NewHSplit(container.NewVScroll(left), right)
}

func openOptionsWindow() {
	w := fyne.CurrentApp().NewWindow("Options")
	w.SetContent(makeScrollTab(w))
	w.Resize(fyne.NewSize(240, 180))

	w.CenterOnScreen()
	w.Show()
}

func manageGames() {

}

func getDefaultGreen() color.Color {
	return color.RGBA{
		R: 65,
		G: 255,
		B: 65,
		A: 255,
	}
}

func GuiMain(ops *Options, dm *GameDefManager) {
	// The steam deck (likely due to it's DPI) has scaling issues with our current version of FYNE
	// To make this smooth, we will scale the UI to make it look nice in gaming mode.
	// Normal linux users can overwrite this.

	// @TODO this makes the window look bad in game mode, needs more investigation
	// Not using this in game mode makes the UI look great, but our mouse X/Y is limited to the upper right
	// quadrant of the UI
	// if runtime.GOOS == "linux" && os.Getenv("FYNE_SCALE") == "" {
	// 	os.Setenv("FYNE_SCALE", "0.25")
	// }

	a := app.New()
	a.SetIcon(fyne.NewStaticResource("Icon", icon))

	w := a.NewWindow("Steam Custom Cloud Uploads")
	w.FullScreen()
	// w.Resize(fyne.NewSize(800, 600))
	w.CenterOnScreen()

	innerContainer := container.NewVBox()
	plainContainer := container.NewVBox(innerContainer)

	list := make([]fyne.CanvasObject, 0)
	syncMap := make(map[string]bool)
	for k, v := range dm.GetGameDefMap() {
		key := k
		list = append(list, widget.NewCheck(v.DisplayName, func(selected bool) {
			syncMap[key] = selected
			plainContainer.Remove(innerContainer)

			if !selected {
				return
			}

			syncpaths, _ := dm.GetSyncpathForGame(key)

			innerContainer = container.NewVBox()

			overallStatus := canvas.NewText("Status: Cloud in Sync", getDefaultGreen())
			overallStatus.TextStyle = fyne.TextStyle{
				Bold: true,
			}
			overallStatus.Alignment = fyne.TextAlignCenter
			innerContainer.Add(overallStatus)

			saveList := make([]*widget.AccordionItem, 0)

			for _, syncpath := range syncpaths {
				files, _ := dm.GetFilesForGame(key, syncpath.Parent)
				for k, v := range files {
					f, err := os.Stat(v.Name)
					if err != nil {
						fmt.Println(err)
					}

					del := widget.NewButton("Delete", func() {
						log.Fatal("Not Implemented")
					})

					sync := widget.NewButton("Sync", func() {
						log.Fatal("Not Implemented")
					})
					sync.Importance = widget.HighImportance

					cloudStatus := canvas.NewText("File in Sync", getDefaultGreen())
					cloudStatus.TextStyle = fyne.TextStyle{
						Bold: true,
					}
					cloudStatus.Alignment = fyne.TextAlignCenter
					itemContainer := container.NewVBox(widget.NewLabel("Save File: "+v.Name),
						widget.NewLabel("Date Modified: "+f.ModTime().String()),
						cloudStatus,
						sync,
						del)
					newItem := widget.NewAccordionItem(k, itemContainer)
					saveList = append(saveList, newItem)

				}
			}

			inn := widget.NewVBox(widget.NewAccordion(saveList...))
			scroll := container.NewVScroll(inn)
			scroll.SetMinSize(fyne.NewSize(500, 500))
			innerContainer.Add(scroll)
			plainContainer.Add(innerContainer)
		}))
	}

	syncButton := widget.NewButton("Sync Selected", func() {
		ops.Gamenames = []string{}
		for k, v := range syncMap {
			if v {
				ops.Gamenames = append(ops.Gamenames, k)
			}
		}

		fmt.Println(ops.Gamenames)
		CliMain(ops, dm)
	})
	syncAllButton := widget.NewButton("Sync All", func() {
		ops.Gamenames = []string{}
		for k := range dm.GetGameDefMap() {
			ops.Gamenames = append(ops.Gamenames, k)
		}

		fmt.Println(ops.Gamenames)
		CliMain(ops, dm)
	})
	manageGamesButton := widget.NewButton("Manage Games", manageGames)
	optionsButton := widget.NewButton("Options", openOptionsWindow)
	hlist := []fyne.CanvasObject{syncAllButton, syncButton, manageGamesButton, optionsButton}
	vlist := list
	horiz := container.NewHScroll(container.NewHBox(hlist...))
	vert := container.NewVScroll(container.NewVBox(vlist...))

	hsplit := container.NewHSplit(vert, plainContainer)
	hsplit.Offset = 0.10

	cont := container.NewVSplit(horiz, hsplit)
	cont.Offset = 0.05

	// Work around for issue https://github.com/DavidDeSimone/CustomSteamCloudUploads/issues/16
	if runtime.GOOS == "darwin" {
		w.SetFixedSize(true)
	}
	w.SetMaster()

	w.SetContent(cont)
	w.ShowAndRun()
}
