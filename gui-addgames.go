package main

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne"
	cont "fyne.io/fyne/container"
	"fyne.io/fyne/widget"
	"github.com/sqweek/dialog"
)

type AddGamesContainer struct {
	dm               GameDefManager
	verticalSplit    *cont.Split
	scroll           *cont.Scroll
	scrollContent    *fyne.Container
	buttonContainer  *fyne.Container
	contentAccordion *widget.Accordion
}

type GameCardContainer struct {
	dm            GameDefManager
	accordionItem *widget.AccordionItem

	displayNameBox   *fyne.Container
	displayNameEntry *widget.Entry

	contentContainer     *fyne.Container
	winEntryContainer    *fyne.Container
	darwinEntryContainer *fyne.Container
	linuxEntryContainer  *fyne.Container

	deleteEntryButton *widget.Button

	key string
	def *GameDef
}

func makeTextEntry(text string, callback func(s string)) *widget.Entry {
	entry := widget.NewEntry()
	entry.Text = text
	entry.OnChanged = callback
	return entry
}

func (g *GameCardContainer) makeCard(path []*Datapath, onRemove func(int, []*Datapath)) *fyne.Container {
	parent := cont.NewVBox()
	innerEntry := cont.NewVBox()
	parent.Add(innerEntry)
	if len(path) > 0 {

		for i, n := range path {
			textEntryFunc := func(s string) {
				separator := os.PathSeparator
				n.Path = s
				entries := strings.Split(s, string(separator))
				n.Parent = entries[len(entries)-1]

				fmt.Printf("Updated %v with values %v\n", n.Path, n.Parent)
			}
			textbox := makeTextEntry(n.Path, textEntryFunc)

			entryPtr, innerPtr := parent, innerEntry
			removeIdx := i

			selectButton := widget.NewButton("Select", func() {
				filename, err := dialog.Directory().Browse()
				if err != nil {
					fmt.Println(err)
				}

				textEntryFunc(filename)
				textbox.Text = filename
				textbox.Refresh()

			})
			removeButton := widget.NewButton("Remove", func() {
				onRemove(removeIdx, path)
				entryPtr.Remove(innerPtr)
			})

			buttonSplit := cont.NewHSplit(selectButton, removeButton)
			line := cont.NewHSplit(textbox, buttonSplit)
			line.Offset = 0.8

			innerEntry.Add(line)
			exts := ""
			for i, e := range n.Exts {
				exts += e
				if i != len(n.Exts)-1 {
					e += ", "
				}
			}

			extsEntry := makeTextEntry(exts, func(s string) {
				entries := strings.Split(s, ",")
				n.Exts = []string{}
				for _, entry := range entries {
					n.Exts = append(n.Exts, strings.TrimSpace(entry))
				}
			})
			innerEntry.Add(widget.NewHBox(widget.NewLabel("Exts: "), extsEntry))

			ignoreListString := ""
			for i, itemToIgnore := range n.Ignore {
				ignoreListString += itemToIgnore
				if i != len(n.Ignore)-1 {
					ignoreListString += ", "
				}
			}

			ignoreEntry := makeTextEntry(ignoreListString, func(s string) {
				entries := strings.Split(s, ",")
				n.Ignore = []string{}
				for _, entry := range entries {
					n.Ignore = append(n.Ignore, strings.TrimSpace(entry))
				}
			})

			innerEntry.Add(cont.NewHBox(widget.NewLabel("Ignored Files (Comma separated)"), ignoreEntry))

			downloadCheck := widget.NewCheck("Download", func(b bool) {
				if b {
					n.NetAuth |= CloudOperationDownload
				} else {
					n.NetAuth &= ^CloudOperationDownload
				}
			})
			downloadCheck.SetChecked(n.NetAuth&CloudOperationDownload != 0)
			uploadCheck := widget.NewCheck("Upload", func(b bool) {
				if b {
					n.NetAuth |= CloudOperationUpload
				} else {
					n.NetAuth &= ^CloudOperationUpload
				}
			})
			uploadCheck.SetChecked(n.NetAuth&CloudOperationUpload != 0)
			deleteCheck := widget.NewCheck("Delete", func(b bool) {
				if b {
					n.NetAuth |= CloudOperationDelete
				} else {
					n.NetAuth &= ^CloudOperationDelete
				}
			})
			deleteCheck.SetChecked(n.NetAuth&CloudOperationDelete != 0)

			innerEntry.Add(cont.NewHBox(downloadCheck, uploadCheck, deleteCheck))
		}
	}

	return parent
}

func removeInx(i int, path []*Datapath) []*Datapath {
	return append(path[:i], path[i+1:]...)
}

func (g *GameCardContainer) makeWinCard() {
	g.winEntryContainer = cont.NewVBox()

	winRemoveFunc := func(i int, d []*Datapath) {
		g.def.WinPath = removeInx(i, d)
	}

	winParent := g.makeCard(g.def.WinPath, winRemoveFunc)
	winParent.Add(widget.NewButton("Add Windows Path", func() {
		g.winEntryContainer.Remove(winParent)
		g.def.WinPath = append(g.def.WinPath, &Datapath{
			NetAuth: CloudOperationAll,
		})
		g.winEntryContainer.Add(g.makeCard(g.def.WinPath, winRemoveFunc))
	}))

	g.winEntryContainer.Add(widget.NewCard("Windows", "", winParent))
	g.contentContainer.Add(g.winEntryContainer)
}

func (g *GameCardContainer) makeLinuxCard() {
	g.linuxEntryContainer = cont.NewVBox()

	linuxRemoveFunc := func(i int, d []*Datapath) {
		g.def.LinuxPath = removeInx(i, d)
	}

	linuxParent := g.makeCard(g.def.LinuxPath, linuxRemoveFunc)
	linuxParent.Add(widget.NewButton("Add Linux Path", func() {
		g.linuxEntryContainer.Remove(linuxParent)
		g.def.LinuxPath = append(g.def.LinuxPath, &Datapath{
			NetAuth: CloudOperationAll,
		})
		g.linuxEntryContainer.Add(g.makeCard(g.def.LinuxPath, linuxRemoveFunc))
	}))

	g.linuxEntryContainer.Add(widget.NewCard("Linux", "", linuxParent))
	g.contentContainer.Add(g.linuxEntryContainer)
}

func (g *GameCardContainer) makeDarwinCard() {
	g.darwinEntryContainer = cont.NewVBox()

	darwinRemoveFunc := func(i int, d []*Datapath) {
		g.def.DarwinPath = removeInx(i, d)
	}

	darwinParent := g.makeCard(g.def.DarwinPath, darwinRemoveFunc)
	darwinParent.Add(widget.NewButton("Add MacOS Path", func() {
		g.darwinEntryContainer.Remove(darwinParent)
		g.def.DarwinPath = append(g.def.DarwinPath, &Datapath{
			NetAuth: CloudOperationAll,
		})
		g.darwinEntryContainer.Add(g.makeCard(g.def.DarwinPath, darwinRemoveFunc))
	}))

	g.darwinEntryContainer.Add(widget.NewCard("MacOS", "", darwinParent))
	g.contentContainer.Add(g.darwinEntryContainer)
}

func (g *AddGamesContainer) makeGameCardEntry(k string, v *GameDef) *GameCardContainer {
	entry := &GameCardContainer{
		dm:  g.dm,
		key: k,
		def: v,
	}

	entry.displayNameEntry = makeTextEntry(v.DisplayName, func(s string) {
		gamedefs := g.dm.GetGameDefMap()
		v.DisplayName = s
		delete(gamedefs, k)
		gamedefs[s] = v

		entry.accordionItem.Title = s
		k = s
	})
	entry.displayNameBox = cont.NewHBox(widget.NewLabel("Display Name: "), entry.displayNameEntry)

	entry.contentContainer = cont.NewVBox(entry.displayNameBox)
	entry.makeWinCard()
	entry.makeDarwinCard()
	entry.makeLinuxCard()
	entry.deleteEntryButton = widget.NewButton("Stop Tracking "+v.DisplayName, func() {
		g.contentAccordion.Remove(entry.accordionItem)
		delete(entry.dm.GetGameDefMap(), k)
		g.scroll.ScrollToTop()

	})
	entry.deleteEntryButton.Importance = widget.HighImportance
	entry.contentContainer.Add(entry.deleteEntryButton)

	entry.accordionItem = widget.NewAccordionItem(v.DisplayName, entry.contentContainer)
	return entry
}

func (g *AddGamesContainer) makeAddGameButton() *widget.Button {
	addGameButton := widget.NewButton("Add Game", func() {
		key := "New Game"
		value := &GameDef{
			DisplayName: "New Game",
		}
		g.dm.GetGameDefMap()[key] = value
		entry := g.makeGameCardEntry(key, value)
		g.contentAccordion.Append(entry.accordionItem)
	})

	addGameButton.Importance = widget.HighImportance
	return addGameButton
}

func (g *AddGamesContainer) makeCloseButton() *widget.Button {
	closeButton := widget.NewButton("Save and Close", func() {
		err := g.dm.CommitUserOverrides()
		if err != nil {
			fmt.Println(err)
		}

		GetViewStack().PopContent()
		GetMainMenu().Refresh()
	})
	return closeButton
}

func MakeAddGamesScreen(dm GameDefManager) fyne.CanvasObject {
	gameScreen := &AddGamesContainer{
		dm:            dm,
		scrollContent: cont.NewVBox(),
	}

	gameList := make([]*widget.AccordionItem, 0)
	for k, v := range dm.GetGameDefMap() {
		entry := gameScreen.makeGameCardEntry(k, v)
		gameList = append(gameList, entry.accordionItem)
	}

	gameScreen.contentAccordion = widget.NewAccordion(gameList...)
	gameScreen.contentAccordion.MultiOpen = true
	gameScreen.scrollContent.Add(gameScreen.contentAccordion)
	gameScreen.scroll = cont.NewVScroll(gameScreen.scrollContent)

	gameScreen.buttonContainer = cont.NewVBox()

	gameScreen.buttonContainer.Add(gameScreen.makeAddGameButton())
	gameScreen.buttonContainer.Add(gameScreen.makeCloseButton())

	gameScreen.verticalSplit = cont.NewVSplit(gameScreen.scroll, gameScreen.buttonContainer)
	gameScreen.verticalSplit.Offset = 0.85
	return gameScreen.verticalSplit
}
