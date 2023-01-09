package main

import (
	"fmt"
	"os"

	"fyne.io/fyne"
	cont "fyne.io/fyne/container"
	"fyne.io/fyne/widget"
)

func MakeOptionsScreen() fyne.CanvasObject {
	container := cont.NewVBox()

	container.Add(widget.NewButton("Delete Cached Cloud Tokens", func() {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			fmt.Println(err)
			return
		}

		err = os.Remove(cacheDir + string(os.PathSeparator) + "token.json")
		if err != nil {
			fmt.Println(err)
		}
	}))

	saveAndCloseButton := widget.NewButton("Close", func() {
		GetViewStack().PopContent()
	})
	container.Add(saveAndCloseButton)

	return container
}
