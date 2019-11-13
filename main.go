package main

import (
	"github.com/mfboulos/oziachbot/bot"
)

func main() {
	oziachBot := bot.InitBot()
	err := oziachBot.Connect()

	if err != nil {
		panic(err)
	}
}
