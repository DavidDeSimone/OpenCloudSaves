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
	"fyne.io/fyne/widget"
)

//go:embed icon.jpg
var icon []byte

func openOptionsWindow() {
	GetViewStack().PushContent(MakeOptionsScreen())
}

func manageGames(dm *GameDefManager) {
	GetViewStack().PushContent(MakeAddGamesScreen(dm))
}

func getDefaultGreen() color.Color {
	return color.RGBA{
		R: 65,
		G: 255,
		B: 65,
		A: 255,
	}
}

func getDefaultRed() color.Color {
	return color.RGBA{
		R: 255,
		G: 65,
		B: 65,
		A: 255,
	}
}

var mainMenu *MainMenuContainer = nil
var syncMap map[string]bool = make(map[string]bool)

func GetMainMenu() *MainMenuContainer {
	if mainMenu == nil {
		mainMenu = &MainMenuContainer{}
	}

	return mainMenu
}

type MainMenuContainer struct {
	dm *GameDefManager

	rootVerticalSplit *widget.SplitContainer
	menuBar           *widget.ScrollContainer

	parentContainer *fyne.Container
	innerContainer  *fyne.Container

	verticalGameScroll *widget.ScrollContainer
	horizSplit         *widget.SplitContainer
}

func (main *MainMenuContainer) RefreshGames() {
	if main.parentContainer != nil {
		main.parentContainer.Remove(main.innerContainer)
	}

	main.innerContainer = container.NewVBox()
	main.parentContainer = container.NewVBox(main.innerContainer)

	list := make([]fyne.CanvasObject, 0)
	srv := GetDefaultService()
	for k, v := range main.dm.GetGameDefMap() {
		key := k
		list = append(list, widget.NewCheck(v.DisplayName, func(selected bool) {
			syncMap[key] = selected
			main.parentContainer.Remove(main.innerContainer)

			if !selected {
				return
			}

			syncpaths, _ := main.dm.GetSyncpathForGame(key)

			main.innerContainer = container.NewVBox()

			overallStatus := canvas.NewText("Status: Cloud in Sync", getDefaultGreen())
			overallStatus.TextStyle = fyne.TextStyle{
				Bold: true,
			}
			overallStatus.Alignment = fyne.TextAlignCenter
			main.innerContainer.Add(overallStatus)

			saveList := make([]*widget.AccordionItem, 0)

			for _, syncpath := range syncpaths {
				localMetaData, err := GetLocalMetadata(syncpath.Path + STEAM_METAFILE)
				if err != nil || localMetaData == nil {
					fmt.Println(err)
					localMetaData = &GameMetadata{
						Version: CURRENT_META_VERSION,
						Gameid:  syncpath.Parent,
						Files:   make(map[string]FileMetadata),
					}
				}

				files, _ := main.dm.GetFilesForGame(key, syncpath.Parent)
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

					cloudStatus := canvas.NewText("", getDefaultGreen())
					// @TODO this is lying if the entries in local meta data are matching and in sync
					// but the remote metadata has new entries
					cloudStatus.Text = "File in Sync"
					cloudStatus.TextStyle = fyne.TextStyle{Bold: true}
					cloudStatus.Alignment = fyne.TextAlignCenter
					metaFile, ok := localMetaData.Files[k]
					if !ok {
						cloudStatus.Text = "File Not in Sync"
						cloudStatus.Color = getDefaultRed()
						overallStatus.Text = "Not all files in Sync"
						overallStatus.Color = getDefaultRed()
					} else {
						syncStatus, err := srv.IsFileInSync(k, v.Name, metaFile.FileId, localMetaData)
						if err != nil || syncStatus != InSync {
							cloudStatus.Text = "File Not in Sync"
							cloudStatus.Color = getDefaultRed()
							overallStatus.Text = "Not all files in Sync"
							overallStatus.Color = getDefaultRed()
						}
					}

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
			main.innerContainer.Add(scroll)
			main.parentContainer.Add(main.innerContainer)

			// @TODO this isn't working right. selecting a game, going into the add game menu and returning breaks the app
			main.RefreshRootView()
		}))
	}

	main.verticalGameScroll = container.NewVScroll(container.NewVBox(list...))
}

func (main *MainMenuContainer) RefreshRootView() {
	if main.horizSplit == nil {
		main.horizSplit = container.NewHSplit(main.verticalGameScroll, main.parentContainer)
		main.horizSplit.Offset = 0.10
	} else {
		main.horizSplit.Leading = main.verticalGameScroll
		main.horizSplit.Trailing = main.parentContainer
	}

	if main.rootVerticalSplit == nil {
		main.rootVerticalSplit = container.NewVSplit(main.menuBar, main.horizSplit)
		main.rootVerticalSplit.Offset = 0.05
	} else {
		main.rootVerticalSplit.Leading = main.menuBar
		main.rootVerticalSplit.Trailing = main.horizSplit
		main.rootVerticalSplit.Refresh()
	}
}

func (main *MainMenuContainer) Refresh() {
	main.RefreshGames()
	main.RefreshRootView()
}

func (main *MainMenuContainer) visualLogging(input chan Message) {
	minSize := main.parentContainer.MinSize()
	parent := container.NewVBox()
	innerBox := container.NewVBox()
	tempScroll := container.NewScroll(innerBox)
	tempScroll.SetMinSize(minSize)
	parent.Add(tempScroll)

	defaultColor := fyne.CurrentApp().Settings().Theme().TextColor()
	root := container.NewVBox(parent)
	GetViewStack().PushContent(root)

	for {
		result := <-input
		if result.Finished {
			fmt.Println("Console Logger Complete...")
			break
		}

		if result.Err != nil {
			msg := canvas.NewText(result.Err.Error(), getDefaultRed())
			msg.TextSize = 14
			innerBox.Add(msg)
		} else {
			msg := canvas.NewText(result.Message, defaultColor)
			msg.TextSize = 10
			innerBox.Add(msg)
		}
	}

	parent.Add(widget.NewButton("Close Logs", func() {
		GetViewStack().PopContent()
	}))
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
	w.Resize(fyne.NewSize(800, 600))
	w.CenterOnScreen()

	main := GetMainMenu()
	main.dm = dm

	main.RefreshGames()

	syncButton := widget.NewButton("Sync Selected", func() {
		ops.Gamenames = []string{}
		for k, v := range syncMap {
			if v {
				ops.Gamenames = append(ops.Gamenames, k)
			}
		}

		logs := make(chan Message, 100)

		fmt.Println(ops.Gamenames)
		go CliMain(ops, dm, logs)
		go main.visualLogging(logs)
	})
	syncAllButton := widget.NewButton("Sync All", func() {
		ops.Gamenames = []string{}
		for k := range dm.GetGameDefMap() {
			ops.Gamenames = append(ops.Gamenames, k)
		}

		logs := make(chan Message, 100)

		fmt.Println(ops.Gamenames)
		go CliMain(ops, dm, logs)
		go main.visualLogging(logs)
	})
	manageGamesButton := widget.NewButton("Manage Games", func() { manageGames(dm) })
	optionsButton := widget.NewButton("Options", openOptionsWindow)
	hlist := []fyne.CanvasObject{syncAllButton, syncButton, manageGamesButton, optionsButton}
	main.menuBar = container.NewHScroll(container.NewHBox(hlist...))

	main.RefreshRootView()

	// Work around for issue https://github.com/DavidDeSimone/CustomSteamCloudUploads/issues/16
	if runtime.GOOS == "darwin" {
		w.SetFixedSize(true)
	}

	v := GetViewStack()
	v.SetMainWindow(w)
	v.PushContent(main.rootVerticalSplit)

	w.SetCloseIntercept(func() {
		dm.CommitUserOverrides()
		os.Exit(0)
	})
	// w.SetContent(cont)
	w.ShowAndRun()
}
