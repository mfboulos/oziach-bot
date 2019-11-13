package hiscores

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// GameMode string representing the type of account
type GameMode string

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
	Defense      skillHiscore
	Strength     skillHiscore
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

// HiscoreAPIError Returned when a player's hiscores aren't available for the given GameMode
type HiscoreAPIError struct {
	Player string
	Mode   GameMode
}

// Enum constants for GameModes
const (
	Normal          GameMode = "normal"
	Ironman         GameMode = "ironman"
	HardcoreIronman GameMode = "hardcore_ironman"
	UltimateIronman GameMode = "ultimate"
)

func (e *UnrankedError) Error() string {
	return "Player is not ranked in this skill/minigame"
}

func (e *HiscoreAPIError) Error() string {
	return fmt.Sprintf("%s is not a(n) %s account", e.Player, e.Mode)
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

func parseCSVHiscores(hiscoreCSV string) Hiscores {
	hiscores := Hiscores{}
	allScores := strings.Fields(hiscoreCSV)

	hiscores.Overall, _ = parseSkillHiscore(allScores[0])
	hiscores.Attack, _ = parseSkillHiscore(allScores[1])
	hiscores.Defense, _ = parseSkillHiscore(allScores[2])
	hiscores.Strength, _ = parseSkillHiscore(allScores[3])
	hiscores.Hitpoints, _ = parseSkillHiscore(allScores[4])
	hiscores.Ranged, _ = parseSkillHiscore(allScores[5])
	hiscores.Prayer, _ = parseSkillHiscore(allScores[6])
	hiscores.Magic, _ = parseSkillHiscore(allScores[7])
	hiscores.Cooking, _ = parseSkillHiscore(allScores[8])
	hiscores.Woodcutting, _ = parseSkillHiscore(allScores[9])
	hiscores.Fletching, _ = parseSkillHiscore(allScores[10])
	hiscores.Fishing, _ = parseSkillHiscore(allScores[11])
	hiscores.Firemaking, _ = parseSkillHiscore(allScores[12])
	hiscores.Crafting, _ = parseSkillHiscore(allScores[13])
	hiscores.Smithing, _ = parseSkillHiscore(allScores[14])
	hiscores.Mining, _ = parseSkillHiscore(allScores[15])
	hiscores.Herblore, _ = parseSkillHiscore(allScores[16])
	hiscores.Agility, _ = parseSkillHiscore(allScores[17])
	hiscores.Thieving, _ = parseSkillHiscore(allScores[18])
	hiscores.Slayer, _ = parseSkillHiscore(allScores[19])
	hiscores.Farming, _ = parseSkillHiscore(allScores[20])
	hiscores.Runecraft, _ = parseSkillHiscore(allScores[21])
	hiscores.Hunter, _ = parseSkillHiscore(allScores[22])
	hiscores.Construction, _ = parseSkillHiscore(allScores[23])

	hiscores.BHHunter, _ = parseMinigameHiscore(allScores[24])
	hiscores.BHRogue, _ = parseMinigameHiscore(allScores[25])

	hiscores.LMS, _ = parseMinigameHiscore(allScores[26])

	cluesOverall, _ := parseMinigameHiscore(allScores[27])
	cluesBeginner, _ := parseMinigameHiscore(allScores[28])
	cluesEasy, _ := parseMinigameHiscore(allScores[29])
	cluesMedium, _ := parseMinigameHiscore(allScores[30])
	cluesHard, _ := parseMinigameHiscore(allScores[31])
	cluesElite, _ := parseMinigameHiscore(allScores[32])
	cluesMaster, _ := parseMinigameHiscore(allScores[33])
	hiscores.Clues = clueHiscores{
		Overall:  cluesOverall,
		Beginner: cluesBeginner,
		Easy:     cluesEasy,
		Medium:   cluesMedium,
		Hard:     cluesHard,
		Elite:    cluesElite,
		Master:   cluesMaster,
	}

	return hiscores
}

// LookupHiscores Looks up a player's hiscores ranked according to the given GameMode
func LookupHiscores(player string, mode GameMode) (Hiscores, error) {
	var gameMode string
	if mode != Normal {
		gameMode = "_" + string(mode)
	} else {
		gameMode = ""
	}

	url := fmt.Sprintf(
		"https://secure.runescape.com/m=hiscore_oldschool%s/index_lite.ws?player=%s",
		gameMode,
		player,
	)

	resp, err := http.Get(url)

	if err != nil {
		return Hiscores{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		body := string(bodyBytes)
		return parseCSVHiscores(body), nil
	}
	
	return Hiscores{}, &HiscoreAPIError{player, mode}
}
