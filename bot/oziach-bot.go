package bot

import (
	"fmt"
	"strings"

	"github.com/gempir/go-twitch-irc"
)

// OziachBot Wrapper class for twitch.Client that encompasses all OziachBot features
type OziachBot struct {
	Client   twitch.Client
	channels []string
}

// HandleMessage Main callback method to wrap all actions on a PRIVMSG
func (bot *OziachBot) HandleMessage(channel string, user twitch.User, message twitch.Message) error {
	switch idx := strings.Index(message.Text, " "); message.Text[:idx] {
	case "!lvl", "!level":
		tokens := strings.SplitN(message.Text, " ", 3)
		if len(tokens) < 3 {
			return &IncorrectFormatError{}
		}

		skillName := tokens[1]
		// Truncate player to 12 characters, max length of an OSRS username
		player := tokens[2][:12]

		go bot.HandleSkillLookup(channel, user, skillName, player)
	case "!total", "!overall":
		tokens := strings.SplitN(message.Text, " ", 2)
		if len(tokens) < 2 {
			return &IncorrectFormatError{}
		}

		skillName := "Overall"
		// Truncate player to 12 characters, max length of an OSRS username
		player := tokens[1][:12]

		go bot.HandleSkillLookup(channel, user, skillName, player)
	}

	return nil
}

// Say Wrapper for Client.Say that prefixes the text with "/me"
func (bot *OziachBot) Say(channel, text string) {
	formattedText := fmt.Sprintf("/me %s", text)
	bot.Client.Say(channel, formattedText)
}
