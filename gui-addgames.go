package main

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne"
	cont "fyne.io/fyne/container"
	"fyne.io/fyne/widget"
)

// @TODO encapsulate this into a class now that the format is more solidified

func makeTextEntry(text string, callback func(s string)) *widget.Entry {
	entry := widget.NewEntry()
	entry.Text = text
	entry.OnChanged = callback
	return entry
}

func drawWindowsCard(winEntryContainer *fyne.Container, winCardContainer *fyne.Container, v *GameDef) *fyne.Container {
	innerEntry := cont.NewVBox()
	winEntryContainer.Add(innerEntry)
	if len(v.WinPath) > 0 {

		for i, n := range v.WinPath {
			textbox := makeTextEntry(n.Path, func(s string) {
				separator := os.PathSeparator
				n.Path = s
				entries := strings.Split(s, string(separator))
				n.Parent = entries[len(entries)-1]

				fmt.Printf("Updated %v with values %v\n", n.Path, n.Parent)
			})

			entryPtr, innerPtr := winEntryContainer, innerEntry
			removeIdx := i
			buttonSplit := widget.NewButton("Remove", func() {
				v.WinPath = append(v.WinPath[:removeIdx], v.WinPath[removeIdx+1:]...)
				entryPtr.Remove(innerPtr)
			})
			line := widget.NewHSplitContainer(textbox, buttonSplit)
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
		}
	}

	return winEntryContainer
}

func MakeAddGamesScreen(dm *GameDefManager) fyne.CanvasObject {
	container := widget.NewVBox()

	gameList := make([]*widget.AccordionItem, 0)
	for kiter, viter := range dm.GetGameDefMap() {
		k, v := kiter, viter
		displayNameEntry := makeTextEntry(v.DisplayName, func(s string) {
			v.DisplayName = s
		})
		displayNameBox := widget.NewHBox(widget.NewLabel("Display Name: "), displayNameEntry)

		innerContainer := widget.NewVBox(
			displayNameBox,
		)

		winEntryContainer := cont.NewVBox()
		winCardContainer := cont.NewVBox(winEntryContainer)
		winUpperContainer := cont.NewVBox(winCardContainer)
		drawWindowsCard(winEntryContainer, winCardContainer, v)
		winUpperContainer.Add(widget.NewButton("Add Windows Path", func() {
			winCardContainer.Remove(winEntryContainer)
			winEntryContainer = cont.NewVBox()
			winCardContainer.Add(winEntryContainer)
			v.WinPath = append(v.WinPath, &Datapath{})
			drawWindowsCard(winEntryContainer, winCardContainer, v)
		}))

		innerContainer.Append(widget.NewCard("Windows", "", winUpperContainer))
		deleteEntryButton := widget.NewButton("Stop Tracking "+v.DisplayName, func() {
			// @TODO show confirmation
			// @TODO remove this entry from visual list
			delete(dm.gamedefs, k)
		})
		deleteEntryButton.Importance = widget.HighImportance
		innerContainer.Append(deleteEntryButton)

		newItem := widget.NewAccordionItem(v.DisplayName, innerContainer)
		gameList = append(gameList, newItem)
	}

	accordion := widget.NewAccordionContainer(gameList...)
	accordion.MultiOpen = true
	container.Append(accordion)
	scroll := widget.NewVScrollContainer(container)

	buttonContainer := widget.NewVBox()
	addGameButton := widget.NewButton("Add Game", func() {
	})
	addGameButton.Importance = widget.HighImportance
	buttonContainer.Append(addGameButton)
	buttonContainer.Append(widget.NewVBox(widget.NewButton("Close", func() {
		err := dm.CommitUserOverrides()
		if err != nil {
			fmt.Println(err)
		}
		GetViewStack().PopContent()
	})))

	vsplit := widget.NewVSplitContainer(scroll, buttonContainer)
	vsplit.Offset = 0.85
	return vsplit
}
