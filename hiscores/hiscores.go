package hiscores

import (
	"strconv"
	"strings"
	//"net/http"
)

type skillHiscore struct {
	Rank  int
	Level int
	Exp   int
}

type minigameHiscore struct {
	Rank  int
	Score int
}

type clueHiscores struct {
	Overall  minigameHiscore
	Beginner minigameHiscore
	Easy     minigameHiscore
	Medium   minigameHiscore
	Hard     minigameHiscore
	Elite    minigameHiscore
	Master   minigameHiscore
}

// Hiscores Model housing all hiscores returned from OSRS Hiscore API
type Hiscores struct {
	Overall      skillHiscore
	Attack       skillHiscore
	Strength     skillHiscore
	Defense      skillHiscore
	Hitpoints    skillHiscore
	Ranged       skillHiscore
	Prayer       skillHiscore
	Magic        skillHiscore
	Cooking      skillHiscore
	Woodcutting  skillHiscore
	Fletching    skillHiscore
	Fishing      skillHiscore
	Firemaking   skillHiscore
	Crafting     skillHiscore
	Smithing     skillHiscore
	Mining       skillHiscore
	Herblore     skillHiscore
	Agility      skillHiscore
	Thieving     skillHiscore
	Slayer       skillHiscore
	Farming      skillHiscore
	Runecraft    skillHiscore
	Hunter       skillHiscore
	Construction skillHiscore

	BHHunter minigameHiscore
	BHRogue  minigameHiscore

	LMS minigameHiscore

	Clues clueHiscores
}

// UnrankedError Returned when a hiscore doesn't exist (player is unranked)
type UnrankedError struct{}

func (e *UnrankedError) Error() string {
	return "Player is not ranked in this skill/minigame"
}

func parseSkillHiscore(hiscore string) (skillHiscore, error) {
	skill := skillHiscore{}

	if vals := strings.Split(hiscore, ","); vals[0] == "-1" {
		skill.Rank, _ = strconv.Atoi(vals[0])
		skill.Level, _ = strconv.Atoi(vals[1])
		skill.Exp, _ = strconv.Atoi(vals[2])
		return skill, nil
	}

	return skill, &UnrankedError{}
}

func parseMinigameHiscore(hiscore string) (minigameHiscore, error) {
	minigame := minigameHiscore{}

	if vals := strings.Split(hiscore, ","); vals[0] == "-1" {
		minigame.Rank, _ = strconv.Atoi(vals[0])
		minigame.Score, _ = strconv.Atoi(vals[1])
		return minigame, nil
	}

	return minigame, &UnrankedError{}
}
