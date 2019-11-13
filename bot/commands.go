package oziachbot

import (
	"strings"

	"github.com/gempir/go-twitch-irc"
)

// HandleSkillLookup parses user message and sends the formatted result of a skill lookup
func (bot *OziachBot) HandleSkillLookup(channel string, user twitch.User, message twitch.Message) {
	tokens := strings.SplitN(message.Text, " ", 3)
	// hit the hiscores api and extract whatever is needed
}
