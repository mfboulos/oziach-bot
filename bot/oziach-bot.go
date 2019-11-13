package oziachbot

import (
	"strings"

	"github.com/gempir/go-twitch-irc"
)

// OziachBot Wrapper class for twitch.Client that encompasses all OziachBot features
type OziachBot struct {
	Client   twitch.Client
	channels []string
}

// HandleMessage Main callback method to wrap all actions on a PRIVMSG
func (bot *OziachBot) HandleMessage(channel string, user twitch.User, message twitch.Message) {
	switch strings.Fields(message.Text)[0] {
	case "!lvl", "!level":
		// go bot.HandleSkillLookup(channel string, user twitch.User, message twitch.Message)
	}
}
