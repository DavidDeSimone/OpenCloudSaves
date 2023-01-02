package main

import (
	"fyne.io/fyne"
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
			pathLabel := widget.NewLabel("Windows")
			pathLabel.Alignment = fyne.TextAlignCenter
			pathLabel.TextStyle = fyne.TextStyle{
				Bold: true,
			}
			innerContainer.Append(pathLabel)

			for _, n := range v.WinPath {
				textbox := makeTextEntry(n.Path, func(s string) {})

				buttonSplit := widget.NewHSplitContainer(widget.NewButton("Copy", func() {}), widget.NewButton("Open", func() {}))
				line := widget.NewHSplitContainer(textbox, buttonSplit)
				line.Offset = 0.8

				innerContainer.Append(line)
				exts := ""
				for i, e := range n.Exts {
					exts += e
					if i != len(n.Exts)-1 {
						e += ", "
					}
				}

				extsEntry := makeTextEntry(exts, func(s string) {})
				innerContainer.Append(widget.NewHBox(widget.NewLabel("Exts: "), extsEntry))
				innerContainer.Append(widget.NewButton("Add Windows Path", func() {}))
			}
		}

		newItem := widget.NewAccordionItem(k, innerContainer)
		gameList = append(gameList, newItem)
	}

	accordion := widget.NewAccordionContainer(gameList...)
	accordion.MultiOpen = true
	container.Append(accordion)
	scroll := widget.NewVScrollContainer(container)

	buttonContainer := widget.NewVBox()
	buttonContainer.Append(widget.NewButton("Add Game", func() {
	}))
	buttonContainer.Append(widget.NewVBox(widget.NewButton("Close", func() {
		GetViewStack().PopContent()
	})))

	vsplit := widget.NewVSplitContainer(scroll, buttonContainer)
	vsplit.Offset = 0.85
	return vsplit
}
