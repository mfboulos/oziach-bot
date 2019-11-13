package bot

import (
	"fmt"

	"github.com/gempir/go-twitch-irc"
	"github.com/mfboulos/oziachbot/hiscores"
)

// IncorrectFormatError Returned when a command invocation is malformed
type IncorrectFormatError struct{}

func (e *IncorrectFormatError) Error() string {
	return "Incorrect format for command"
}

// HandleSkillLookup parses user message and sends the formatted result of a skill lookup
func (bot *OziachBot) HandleSkillLookup(channel string, user twitch.User, skillName, player string) error {
	mode := hiscores.Normal
	playerHiscores, err := hiscores.LookupHiscores(player, mode)

	// All players should be in normal hiscores. If they are not, they are either unranked
	// or they do not exist
	if err != nil {
		bot.Say(channel, fmt.Sprintf("@%s Could not find player %s", user.DisplayName, player))
		return err
	}

	ironmanHiscores, err2 := hiscores.LookupHiscores(player, hiscores.Ironman)
	if err2 == nil {
		hardcoreHiscores, err3 := hiscores.LookupHiscores(player, hiscores.HardcoreIronman)
		ultimateHiscores, err4 := hiscores.LookupHiscores(player, hiscores.UltimateIronman)

		// To determine what kind of ironman we have, first we check to see if there is a
		// hiscore under that GameMode. If the hiscores in that GameMode and GameMode.Ironman
		// match, the player is in that GameMode
		if err3 == nil && hardcoreHiscores == ironmanHiscores {
			playerHiscores = hardcoreHiscores
			mode = hiscores.HardcoreIronman
		} else if err4 == nil && ultimateHiscores == ironmanHiscores {
			playerHiscores = ultimateHiscores
			mode = hiscores.UltimateIronman
		} else {
			playerHiscores = ironmanHiscores
			mode = hiscores.Ironman
		}
	}

	skill, err5 := playerHiscores.GetSkillHiscoreFromName(skillName)

	// If the name doesn't map to a skill, the bot silently fails to retrieve it
	if err5 != nil {
		return err5
	}

	bot.Client.Say(channel, fmt.Sprintf(
		"/me @%s - %s, %s level: %d, Rank (%s): %d, Exp: %d",
		user.DisplayName,
		player,
		skill.Name,
		skill.Level,
		mode.Name,
		skill.Rank,
		skill.Exp,
	))

	return nil
}
