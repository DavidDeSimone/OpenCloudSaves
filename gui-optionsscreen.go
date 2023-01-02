package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/widget"
)

func MakeOptionsScreen() fyne.CanvasObject {
	container := widget.NewVBox()

	container.Append(widget.NewVBox(widget.NewButton("Close", func() {
		GetViewStack().PopContent()
	})))

	return container
}
