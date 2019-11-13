package main

import (
	"fmt"
	"os"

	"github.com/gempir/go-twitch-irc"
)

func main() {
	client := twitch.NewClient("OziachBot", fmt.Sprintf("oauth:%s", os.Getenv("OZIACH_AUTH")))

	// Here we perform all the setup needed for the Client:
	// * Rooms to join
	// * Callback setup

	// TODO: wrap client in bot object and complete all of the above

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
