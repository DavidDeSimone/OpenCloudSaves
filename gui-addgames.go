package main

import (
	"fmt"

	"fyne.io/fyne"
	cont "fyne.io/fyne/container"
	"fyne.io/fyne/widget"
)

func makeTextEntry(text string, callback func(s string)) *widget.Entry {
	entry := widget.NewEntry()
	entry.Text = text
	entry.OnChanged = callback
	return entry
}

func MakeAddGamesScreen(dm *GameDefManager) fyne.CanvasObject {
	container := widget.NewVBox()

	gameList := make([]*widget.AccordionItem, 0)
	for k, v := range dm.GetGameDefMap() {
		displayNameEntry := makeTextEntry(v.DisplayName, func(s string) {})
		displayNameBox := widget.NewHBox(widget.NewLabel("Display Name: "), displayNameEntry)

		innerContainer := widget.NewVBox(
			displayNameBox,
		)

		if len(v.WinPath) > 0 {
			entryContainer := cont.NewVBox()
			cardContainer := cont.NewVBox(entryContainer)

			for _, n := range v.WinPath {
				textbox := makeTextEntry(n.Path, func(s string) {})

				c, e := cardContainer, entryContainer
				buttonSplit := widget.NewHSplitContainer(widget.NewButton("Remove", func() {
					c.Remove(e)
				}), widget.NewButton("Open", func() {}))
				line := widget.NewHSplitContainer(textbox, buttonSplit)
				line.Offset = 0.8

				entryContainer.Add(line)
				exts := ""
				for i, e := range n.Exts {
					exts += e
					if i != len(n.Exts)-1 {
						e += ", "
					}
				}

				extsEntry := makeTextEntry(exts, func(s string) { fmt.Println(s) })
				entryContainer.Add(widget.NewHBox(widget.NewLabel("Exts: "), extsEntry))
			}

			cardContainer.Add(widget.NewButton("Add Windows Path", func() {}))
			innerContainer.Append(widget.NewCard("Windows", "", cardContainer))
		}

		deleteEntryButton := widget.NewButton("Stop Tracking "+v.DisplayName, func() {})
		deleteEntryButton.Importance = widget.HighImportance
		innerContainer.Append(deleteEntryButton)

		newItem := widget.NewAccordionItem(k, innerContainer)
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
		GetViewStack().PopContent()
	})))

	vsplit := widget.NewVSplitContainer(scroll, buttonContainer)
	vsplit.Offset = 0.85
	return vsplit
}
