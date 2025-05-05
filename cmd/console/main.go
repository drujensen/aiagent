package main

import (
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	menu := tview.NewList()
	menu.AddItem("New Chat", "", '1', nil)
	menu.AddItem("Existing Chat", "", '2', nil)
	menu.AddItem("Settings", "", '3', nil)
	menu.AddItem("Exit", "", 'q', func() {
		app.Stop()
	})
	menu.ShowSecondaryText(false).SetBorder(true).SetTitle("AI Agent Menu")
	menu.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		switch index {
		case 0:
			// Start New Chat
		case 1:
			// Load Existing Chat
		case 2:
			// Settings
		default:
			// Do nothing
		}
	})
	if err := app.SetRoot(menu, true).SetFocus(menu).Run(); err != nil {
		panic(err)
	}
}
