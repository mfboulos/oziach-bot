package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/gempir/go-twitch-irc"
)

// OziachBot Wrapper class for twitch.Client that encompasses all OziachBot features
type OziachBot struct {
	Client   *twitch.Client
	channels []string
}

// InitBot Initalizes OziachBot with callbacks and channels it needs to join
func InitBot() OziachBot {
	client := twitch.NewClient("OziachBot", fmt.Sprintf("oauth:%s", os.Getenv("OZIACH_AUTH")))

	// Here we perform all the setup needed for the Client:
	// * Rooms to join
	// * Callback setup
	channels := []string{"solisrs"} // TODO: delete after quick test
	bot := OziachBot{client, channels}

	client.OnNewMessage(bot.HandleMessage)

	for _, channel := range channels {
		client.Join(channel)
	}

	return bot
}

// Connect Connects OziachBot to Twitch IRC
func (bot *OziachBot) Connect() error {
	return bot.Client.Connect()
}

// HandleMessage Main callback method to wrap all actions on a PRIVMSG
func (bot *OziachBot) HandleMessage(channel string, user twitch.User, message twitch.Message) {
	idx := strings.Index(message.Text, " ")
	if idx == -1 {
		idx = len(message.Text)
	}

	switch message.Text[:idx] {
	case "!lvl", "!level":
		tokens := strings.SplitN(message.Text, " ", 3)
		if len(tokens) < 3 {
			break
		}

		skillName := tokens[1]
		// Truncate player to 12 characters, max length of an OSRS username
		player := tokens[2][:12]

		go bot.HandleSkillLookup(channel, user, skillName, player)
	case "!total", "!overall":
		tokens := strings.SplitN(message.Text, " ", 2)
		if len(tokens) < 2 {
			break
		}

		skillName := "Overall"
		// Truncate player to 12 characters, max length of an OSRS username
		player := tokens[1][:12]

		go bot.HandleSkillLookup(channel, user, skillName, player)
	}
}

// Say Wrapper for Client.Say that prefixes the text with "/me"
func (bot *OziachBot) Say(channel, text string) {
	formattedText := fmt.Sprintf("/me %s", text)
	bot.Client.Say(channel, formattedText)
}
