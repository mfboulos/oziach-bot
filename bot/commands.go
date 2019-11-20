package bot

import (
	"fmt"

	"github.com/dustin/go-humanize"
)

// IncorrectFormatError Returned when a command invocation is malformed
type IncorrectFormatError struct{}

func (e *IncorrectFormatError) Error() string {
	return "Incorrect format for command"
}

// HandleSkillLookup parses user message and sends the formatted result of a skill lookup
func (bot *OziachBot) HandleSkillLookup(channel, user, skillName, player string) error {
	playerHiscores, mode, err := bot.HiscoreAPI.LookupHiscores(player)
	if err != nil {
		bot.Say(channel, fmt.Sprintf("@%s Could not find player %s", user, player))
		return err
	}

	name, skill, err := playerHiscores.GetSkillHiscoreFromName(skillName)

	// If the name doesn't map to a skill, the bot silently fails to retrieve it
	if err != nil {
		return err
	}

	bot.TwitchClient.Say(channel, fmt.Sprintf(
		"/me @%s - %s | %s level: %s | Rank (%s): %s | Exp: %s",
		user,
		player,
		name,
		humanize.Comma(int64(skill.Level)),
		mode.Name,
		humanize.Comma(int64(skill.Rank)),
		humanize.Comma(int64(skill.Exp)),
	))

	return nil
}
