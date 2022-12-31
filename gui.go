package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/driver/desktop"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

//go:embed icon.jpg
var icon []byte

func windowScreen(_ fyne.Window) fyne.CanvasObject {
	windowGroup := container.NewVBox(
		widget.NewButton("New window", func() {
			w := fyne.CurrentApp().NewWindow("Hello")
			w.SetContent(widget.NewLabel("Hello World!"))
			w.Show()
		}),
		widget.NewButton("Fixed size window", func() {
			w := fyne.CurrentApp().NewWindow("Fixed")
			w.SetContent(fyne.NewContainerWithLayout(layout.NewCenterLayout(), widget.NewLabel("Hello World!")))

			w.Resize(fyne.NewSize(240, 180))
			w.SetFixedSize(true)
			w.Show()
		}),
		widget.NewButton("Centered window", func() {
			w := fyne.CurrentApp().NewWindow("Central")
			w.SetContent(fyne.NewContainerWithLayout(layout.NewCenterLayout(), widget.NewLabel("Hello World!")))

			w.CenterOnScreen()
			w.Show()
		}))

	drv := fyne.CurrentApp().Driver()
	if drv, ok := drv.(desktop.Driver); ok {
		windowGroup.Objects = append(windowGroup.Objects,
			widget.NewButton("Splash Window (only use on start)", func() {
				w := drv.CreateSplashWindow()
				w.SetContent(widget.NewLabelWithStyle("Hello World!\n\nMake a splash!",
					fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
				w.Show()

				go func() {
					time.Sleep(time.Second * 3)
					w.Close()
				}()
			}))
	}

	otherGroup := widget.NewCard("Other", "",
		widget.NewButton("Notification", func() {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Fyne Demo",
				Content: "Testing notifications...",
			})
		}))

	return container.NewVBox(widget.NewCard("Windows", "", windowGroup), otherGroup)
}

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

func openOptionsWindow() {
	w := fyne.CurrentApp().NewWindow("Options")
	w.SetContent(makeScrollTab(w))
	w.Resize(fyne.NewSize(240, 180))

	w.CenterOnScreen()
	w.Show()
}

func manageGames() {

}

func GuiMain(ops *Options, dm *GameDefManager) {
	a := app.New()
	a.SetIcon(fyne.NewStaticResource("Icon", icon))

	w := a.NewWindow("Steam Custom Cloud Uploads")
	w.Resize(fyne.NewSize(500, 500))
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
			saveList := make([]*widget.AccordionItem, 0)

			for _, syncpath := range syncpaths {
				files, _ := dm.GetFilesForGame(key, syncpath.Parent)
				for k, v := range files {
					f, err := os.Stat(v.Name)
					if err != nil {
						fmt.Println(err)
					}

					itemContainer := container.NewVBox(widget.NewLabel("Save File: "+v.Name),
						widget.NewLabel("Date Modified: "+f.ModTime().String()),
						widget.NewButton("Sync", func() {

						}),
						widget.NewButton("Delete", func() {

						}))
					newItem := widget.NewAccordionItem(k, itemContainer)
					saveList = append(saveList, newItem)

				}
			}

			innerContainer.Add(widget.NewAccordion(saveList...))
			plainContainer.Add(innerContainer)
			plainContainer.Resize(fyne.NewSize(400, 400))
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
		for k, v := range syncMap {
			if v {
				ops.Gamenames = append(ops.Gamenames, k)
			}
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

	cont := container.NewAdaptiveGrid(2,
		container.NewBorder(horiz, nil, nil, nil, vert),
		plainContainer)

	w.SetContent(cont)
	w.ShowAndRun()
}
