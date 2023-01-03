package main

import (
	"fmt"
	"log"

	"fyne.io/fyne"
)

type stack[V any] []V

func (s *stack[V]) Push(v V) int {
	*s = append(*s, v)
	return len(*s)
}
func (s *stack[V]) Last() V {
	l := len(*s)

	if l == 0 {
		log.Fatal("Empty Stack")
	}

	last := (*s)[l-1]
	return last
}

func (s *stack[V]) Pop() V {
	removed := (*s).Last()
	*s = (*s)[:len(*s)-1]

	return removed
}

type ViewStack struct {
	rootWindow fyne.Window
	viewStack  stack[fyne.CanvasObject]
}

var applicationViewStack *ViewStack

func GetViewStack() *ViewStack {
	if applicationViewStack == nil {
		applicationViewStack = &ViewStack{}
	}

	return applicationViewStack
}

func (v *ViewStack) SetMainWindow(w fyne.Window) {
	v.rootWindow = w
}

func (v *ViewStack) PushContent(content fyne.CanvasObject) {
	v.viewStack.Push(content)
	v.rootWindow.SetContent(content)
}

func (v *ViewStack) SwapRoot(content fyne.CanvasObject) {
	if len(v.viewStack) > 0 {
		v.viewStack[0] = content
		v.rootWindow.SetContent(content)
		content.Show()
	}
}

func (v *ViewStack) PopContent() {
	if len(v.viewStack) == 1 {
		fmt.Println("Attempting to pop from root view.")
		return
	}

	v.viewStack.Pop()
	v.viewStack.Last().Show()
	v.rootWindow.SetContent(v.viewStack.Last())
}
