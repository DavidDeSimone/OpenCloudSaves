package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"os"
	"runtime"
	"sort"

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

func manageGames(dm GameDefManager) {
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
	dm GameDefManager

	rootVerticalSplit *container.Split
	menuBar           *container.Scroll

	parentContainer *fyne.Container
	innerContainer  *fyne.Container

	verticalGameScroll *container.Scroll
	horizSplit         *container.Split
}

func (main *MainMenuContainer) RefreshGames() {
	defaultText := "Select a game to view saves files!"
	defaultLabel := widget.NewLabel(defaultText)
	defaultLabel.Alignment = fyne.TextAlignCenter
	main.innerContainer = container.NewVBox()
	main.parentContainer = container.NewVBox(defaultLabel, main.innerContainer)

	defMap := main.dm.GetGameDefMap()

	keys := make([]string, 0)
	for k := range defMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := make([]fyne.CanvasObject, 0)
	for _, k := range keys {
		key := k
		value := defMap[key]
		list = append(list, widget.NewCheck(value.DisplayName, func(selected bool) {
			if selected {
				defaultLabel.Text = "Currently Viewing: " + value.DisplayName
			} else {
				defaultLabel.Text = defaultText
			}

			syncMap[key] = selected
			main.parentContainer.Remove(main.innerContainer)

			if !selected {
				return
			}

			syncpaths, _ := main.dm.GetSyncpathForGame(key)

			main.innerContainer = container.NewVBox()
			saveList := make([]*widget.AccordionItem, 0)

			for _, syncpath := range syncpaths {
				files, _ := main.dm.GetFilesForGame(key, syncpath.Parent)
				for k, v := range files {
					f, err := os.Stat(v.Name)
					if err != nil {
						fmt.Println(err)
					}

					itemContainer := container.NewVBox(widget.NewLabel("Save File: "+v.Name),
						widget.NewLabel("Date Modified: "+f.ModTime().String()),
						widget.NewLabel(fmt.Sprintf("Size: %vMB", f.Size()/(1024*1024))),
					)
					newItem := widget.NewAccordionItem(k, itemContainer)
					saveList = append(saveList, newItem)

				}
			}

			statusLabel := canvas.NewText("Current Status: Unknown", fyne.CurrentApp().Settings().Theme().TextColor())
			statusLabel.Alignment = fyne.TextAlignCenter
			statusButton := widget.NewButton("Check Sync Status", func() {
				statusLabel.Text = "Checking Status...."
				statusLabel.Color = fyne.CurrentApp().Settings().Theme().TextColor()
				srv := GetDefaultService()
				outOfSync := false
				for _, syncpath := range syncpaths {
					files, _ := main.dm.GetFilesForGame(key, syncpath.Parent)
					localMetaData, err := GetLocalMetadata(syncpath.Path+STEAM_METAFILE, GetDefaultLocalFs())
					noLocal := false
					noRemote := false
					if err != nil || localMetaData == nil {
						noLocal = true
					}

					var metadata *GameMetadata = nil
					if localMetaData != nil {
						metadata, err = srv.GetMetaData(localMetaData.ParentId, STEAM_METAFILE)
						if err != nil || metadata == nil {
							noRemote = true
						}
					}

					if noLocal || noRemote {
						outOfSync = true
						break
					}

					for k := range metadata.Files {
						_, ok := localMetaData.Files[k]
						if !ok {
							outOfSync = true
							break
						}
					}

					for k := range files {
						local, localOk := localMetaData.Files[k]
						remote, remoteOk := metadata.Files[k]

						if localOk && remoteOk {
							if local.Sha256 != remote.Sha256 {
								outOfSync = true
								break
							}
						} else {
							outOfSync = true
							break
						}
					}

				}

				if outOfSync {
					statusLabel.Text = "Current Status: Out of Sync"
					statusLabel.Color = getDefaultRed()
					statusLabel.TextSize = 16
				} else {
					statusLabel.Text = "Current Status: In Sync"
					statusLabel.Color = getDefaultGreen()
					statusLabel.TextSize = 16
				}
			})
			statusButton.Importance = widget.HighImportance
			labelSplit := container.NewHSplit(statusButton, statusLabel)
			labelSplit.Offset = 0.3

			inn := widget.NewVBox(
				labelSplit,
				widget.NewAccordion(saveList...))
			scroll := container.NewVScroll(inn)
			scroll.SetMinSize(fyne.NewSize(500, 500))
			main.innerContainer.Add(scroll)
			main.parentContainer.Add(main.innerContainer)
		}))
	}
	main.verticalGameScroll = container.NewVScroll(container.NewVBox(list...))
}

func (main *MainMenuContainer) RefreshRootView() {
	main.horizSplit = container.NewHSplit(main.verticalGameScroll, main.parentContainer)
	main.horizSplit.Offset = 0.10

	main.rootVerticalSplit = container.NewVSplit(main.menuBar, main.horizSplit)
	main.rootVerticalSplit.Offset = 0.05
	main.parentContainer.Add(main.innerContainer)
	GetViewStack().SetRoot(main.rootVerticalSplit)
}

func (main *MainMenuContainer) Refresh() {
	syncMap = make(map[string]bool)
	main.RefreshGames()
	main.RefreshRootView()
}

func (main *MainMenuContainer) visualLogging(input chan Message, cancel chan Cancellation) {
	minSize := main.parentContainer.MinSize()
	parent := container.NewVBox()
	innerBox := container.NewVBox()
	tempScroll := container.NewScroll(innerBox)
	tempScroll.SetMinSize(minSize)
	parent.Add(tempScroll)

	defaultColor := fyne.CurrentApp().Settings().Theme().TextColor()
	root := container.NewVBox(parent)
	GetViewStack().PushContent(root)

	parent.Add(widget.NewButton("Cancel", func() {
		cancel <- Cancellation{
			ShouldCancel: true,
		}

		GetViewStack().PopContent()
	}))

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

		GetViewStack().PeekContent().Refresh()
		tempScroll.ScrollToBottom()
	}

	parent.Add(widget.NewButton("Close Logs", func() {
		GetViewStack().PopContent()
	}))
}

func GuiMain(ops *Options, dm GameDefManager) {
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

	syncButton := widget.NewButton("Sync Selected Games", func() {
		ops.Gamenames = []string{}
		for k, v := range syncMap {
			if v {

				ops.Gamenames = append(ops.Gamenames, k)
			}
		}
		channels := &ChannelProvider{
			logs:   make(chan Message, 100),
			cancel: make(chan Cancellation, 1),
		}

		fmt.Println(ops.Gamenames)
		go CliMain(ops, dm, channels, GetDefaultLocalFs())
		go main.visualLogging(channels.logs, channels.cancel)
	})
	syncAllButton := widget.NewButton("Sync All Games", func() {
		ops.Gamenames = []string{}
		for k := range dm.GetGameDefMap() {
			ops.Gamenames = append(ops.Gamenames, k)
		}

		channels := &ChannelProvider{
			logs:   make(chan Message, 100),
			cancel: make(chan Cancellation, 1),
		}

		fmt.Println(ops.Gamenames)
		go CliMain(ops, dm, channels, GetDefaultLocalFs())
		go main.visualLogging(channels.logs, channels.cancel)
	})
	manageGamesButton := widget.NewButton("Manage Games", func() { manageGames(dm) })
	optionsButton := widget.NewButton("Options", openOptionsWindow)
	hlist := []fyne.CanvasObject{syncButton, syncAllButton, manageGamesButton, optionsButton}
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
