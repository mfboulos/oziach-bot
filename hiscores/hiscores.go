package hiscores

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// GameMode struct representing the type of account
type GameMode struct {
	Name         string
	urlComponent string
}

// SkillHiscore struct representing the hiscore of a single skill
type SkillHiscore struct {
	Name  string
	Rank  int
	Level int
	Exp   int
}

// MinigameHiscore struct representing the hiscore of anything that's not a skill
type MinigameHiscore struct {
	Name  string
	Rank  int
	Score int
}

type clueHiscores struct {
	Overall  MinigameHiscore
	Beginner MinigameHiscore
	Easy     MinigameHiscore
	Medium   MinigameHiscore
	Hard     MinigameHiscore
	Elite    MinigameHiscore
	Master   MinigameHiscore
}

// Hiscores Model housing all hiscores returned from OSRS Hiscore API
type Hiscores struct {
	Overall      SkillHiscore
	Attack       SkillHiscore
	Defense      SkillHiscore
	Strength     SkillHiscore
	Hitpoints    SkillHiscore
	Ranged       SkillHiscore
	Prayer       SkillHiscore
	Magic        SkillHiscore
	Cooking      SkillHiscore
	Woodcutting  SkillHiscore
	Fletching    SkillHiscore
	Fishing      SkillHiscore
	Firemaking   SkillHiscore
	Crafting     SkillHiscore
	Smithing     SkillHiscore
	Mining       SkillHiscore
	Herblore     SkillHiscore
	Agility      SkillHiscore
	Thieving     SkillHiscore
	Slayer       SkillHiscore
	Farming      SkillHiscore
	Runecraft    SkillHiscore
	Hunter       SkillHiscore
	Construction SkillHiscore

	BHHunter MinigameHiscore
	BHRogue  MinigameHiscore

	LMS MinigameHiscore

	Clues clueHiscores
}

// UnrankedError Returned when a hiscore doesn't exist (player is unranked)
type UnrankedError struct{}

// HiscoreAPIError Returned when a player's hiscores aren't available for the given GameMode
type HiscoreAPIError struct {
	Player string
	Mode   GameMode
}

// Enumerated GameMode values
var (
	Normal          GameMode = GameMode{"Normal", ""}
	Ironman         GameMode = GameMode{"Ironman", "_ironman"}
	HardcoreIronman GameMode = GameMode{"Hardcore Ironman", "_hardcore_ironman"}
	UltimateIronman GameMode = GameMode{"Ultimate Ironman", "_ultimate"}
)

func (e *UnrankedError) Error() string {
	return "Player is not ranked in this skill/minigame"
}

func (e *HiscoreAPIError) Error() string {
	return fmt.Sprintf("%s is not a(n) %s account", e.Player, e.Mode)
}

// SameScores Returns true if the hiscores represent the same account, false otherwise
func SameScores(h1, h2 Hiscores) bool {
	// If any of these cases hit, it'll break out of the switch
	//
	// I don't like it either, but not having multiline or statements and the object
	// property structure of Hiscores forces us to check equality independently
	switch {
	case h1.Overall.Exp != h2.Overall.Exp:
	case h1.Overall.Exp != h2.Overall.Exp:
	case h1.Attack.Exp != h2.Attack.Exp:
	case h1.Defense.Exp != h2.Defense.Exp:
	case h1.Strength.Exp != h2.Strength.Exp:
	case h1.Hitpoints.Exp != h2.Hitpoints.Exp:
	case h1.Ranged.Exp != h2.Ranged.Exp:
	case h1.Prayer.Exp != h2.Prayer.Exp:
	case h1.Magic.Exp != h2.Magic.Exp:
	case h1.Cooking.Exp != h2.Cooking.Exp:
	case h1.Woodcutting.Exp != h2.Woodcutting.Exp:
	case h1.Fletching.Exp != h2.Fletching.Exp:
	case h1.Fishing.Exp != h2.Fishing.Exp:
	case h1.Firemaking.Exp != h2.Firemaking.Exp:
	case h1.Crafting.Exp != h2.Crafting.Exp:
	case h1.Smithing.Exp != h2.Smithing.Exp:
	case h1.Mining.Exp != h2.Mining.Exp:
	case h1.Herblore.Exp != h2.Herblore.Exp:
	case h1.Agility.Exp != h2.Agility.Exp:
	case h1.Thieving.Exp != h2.Thieving.Exp:
	case h1.Slayer.Exp != h2.Slayer.Exp:
	case h1.Farming.Exp != h2.Farming.Exp:
	case h1.Runecraft.Exp != h2.Runecraft.Exp:
	case h1.Hunter.Exp != h2.Hunter.Exp:
	case h1.Construction.Exp != h2.Construction.Exp:
	default:
		return true
	}
	return false
}

// GetSkillHiscoreFromName maps string name to a specific hiscore
func (hiscores Hiscores) GetSkillHiscoreFromName(name string) (SkillHiscore, error) {
	// Skill name and alias mapping to individual skill hiscores
	skillMap := map[string]SkillHiscore{
		"overall":      hiscores.Overall,
		"total":        hiscores.Overall,
		"attack":       hiscores.Attack,
		"atk":          hiscores.Attack,
		"defense":      hiscores.Defense,
		"def":          hiscores.Defense,
		"strength":     hiscores.Strength,
		"str":          hiscores.Strength,
		"hitpoints":    hiscores.Hitpoints,
		"hp":           hiscores.Hitpoints,
		"ranged":       hiscores.Ranged,
		"range":        hiscores.Ranged,
		"ranging":      hiscores.Ranged,
		"prayer":       hiscores.Prayer,
		"pray":         hiscores.Prayer,
		"magic":        hiscores.Magic,
		"mage":         hiscores.Magic,
		"magician":     hiscores.Magic,
		"cooking":      hiscores.Cooking,
		"cook":         hiscores.Cooking,
		"woodcutting":  hiscores.Woodcutting,
		"woodcut":      hiscores.Woodcutting,
		"wc":           hiscores.Woodcutting,
		"fletching":    hiscores.Fletching,
		"fletch":       hiscores.Fletching,
		"fishing":      hiscores.Fishing,
		"fish":         hiscores.Fishing,
		"firemaking":   hiscores.Firemaking,
		"fm":           hiscores.Firemaking,
		"crafting":     hiscores.Crafting,
		"craft":        hiscores.Crafting,
		"smithing":     hiscores.Smithing,
		"smith":        hiscores.Smithing,
		"mining":       hiscores.Mining,
		"mine":         hiscores.Mining,
		"herblore":     hiscores.Herblore,
		"herb":         hiscores.Herblore,
		"agility":      hiscores.Agility,
		"agil":         hiscores.Agility,
		"thieving":     hiscores.Thieving,
		"thieve":       hiscores.Thieving,
		"thiev":        hiscores.Thieving,
		"slayer":       hiscores.Slayer,
		"slay":         hiscores.Slayer,
		"farming":      hiscores.Farming,
		"farm":         hiscores.Farming,
		"kkona":        hiscores.Farming,
		"runecraft":    hiscores.Runecraft,
		"rc":           hiscores.Runecraft,
		"hunter":       hiscores.Hunter,
		"hunting":      hiscores.Hunter,
		"hunt":         hiscores.Hunter,
		"construction": hiscores.Construction,
		"con":          hiscores.Construction,
	}

	if skillHiscore, ok := skillMap[strings.ToLower(name)]; ok {
		return skillHiscore, nil
	}

	return hiscores.Overall, errors.New("Could not map name to skill")
}

func parseSkillHiscore(name, hiscore string) (SkillHiscore, error) {
	skill := SkillHiscore{Name: name}

	// When a player is unranked, the result is -1,-1
	if vals := strings.Split(hiscore, ","); vals[0] != "-1" {
		skill.Rank, _ = strconv.Atoi(vals[0])
		skill.Level, _ = strconv.Atoi(vals[1])
		skill.Exp, _ = strconv.Atoi(vals[2])
		return skill, nil
	}

	return skill, &UnrankedError{}
}

func parseMinigameHiscore(name, hiscore string) (MinigameHiscore, error) {
	minigame := MinigameHiscore{Name: name}

	// When a player is unranked, the result is -1,-1
	if vals := strings.Split(hiscore, ","); vals[0] != "-1" {
		minigame.Rank, _ = strconv.Atoi(vals[0])
		minigame.Score, _ = strconv.Atoi(vals[1])
		return minigame, nil
	}

	return minigame, &UnrankedError{}
}

func parseCSVHiscores(hiscoreCSV string) Hiscores {
	hiscores := Hiscores{}
	allScores := strings.Fields(hiscoreCSV)

	// Skill mappings
	hiscores.Overall, _ = parseSkillHiscore("Overall", allScores[0])
	hiscores.Attack, _ = parseSkillHiscore("Attack", allScores[1])
	hiscores.Defense, _ = parseSkillHiscore("Defense", allScores[2])
	hiscores.Strength, _ = parseSkillHiscore("Strength", allScores[3])
	hiscores.Hitpoints, _ = parseSkillHiscore("Hitpoints", allScores[4])
	hiscores.Ranged, _ = parseSkillHiscore("Ranged", allScores[5])
	hiscores.Prayer, _ = parseSkillHiscore("Prayer", allScores[6])
	hiscores.Magic, _ = parseSkillHiscore("Magic", allScores[7])
	hiscores.Cooking, _ = parseSkillHiscore("Cooking", allScores[8])
	hiscores.Woodcutting, _ = parseSkillHiscore("Woodcutting", allScores[9])
	hiscores.Fletching, _ = parseSkillHiscore("Fletching", allScores[10])
	hiscores.Fishing, _ = parseSkillHiscore("Fishing", allScores[11])
	hiscores.Firemaking, _ = parseSkillHiscore("Firemaking", allScores[12])
	hiscores.Crafting, _ = parseSkillHiscore("Crafting", allScores[13])
	hiscores.Smithing, _ = parseSkillHiscore("Smithing", allScores[14])
	hiscores.Mining, _ = parseSkillHiscore("Mining", allScores[15])
	hiscores.Herblore, _ = parseSkillHiscore("Herblore", allScores[16])
	hiscores.Agility, _ = parseSkillHiscore("Agility", allScores[17])
	hiscores.Thieving, _ = parseSkillHiscore("Thieving", allScores[18])
	hiscores.Slayer, _ = parseSkillHiscore("Slayer", allScores[19])
	hiscores.Farming, _ = parseSkillHiscore("Farming", allScores[20])
	hiscores.Runecraft, _ = parseSkillHiscore("Runecraft", allScores[21])
	hiscores.Hunter, _ = parseSkillHiscore("Hunter", allScores[22])
	hiscores.Construction, _ = parseSkillHiscore("Construction", allScores[23])

	// Bounty Hunter mappings
	hiscores.BHHunter, _ = parseMinigameHiscore("Bounty Hunter - Hunter", allScores[24])
	hiscores.BHRogue, _ = parseMinigameHiscore("Bounty Hunter - Rogue", allScores[25])

	// LMS mappings
	hiscores.LMS, _ = parseMinigameHiscore("LMS", allScores[26])

	// Clue mappings
	cluesOverall, _ := parseMinigameHiscore("Clue Scrolls - Overall", allScores[27])
	cluesBeginner, _ := parseMinigameHiscore("Clue Scrolls - Beginner", allScores[28])
	cluesEasy, _ := parseMinigameHiscore("Clue Scrolls - Easy", allScores[29])
	cluesMedium, _ := parseMinigameHiscore("Clue Scrolls - Medium", allScores[30])
	cluesHard, _ := parseMinigameHiscore("Clue Scrolls - Hard", allScores[31])
	cluesElite, _ := parseMinigameHiscore("Clue Scrolls - Elite", allScores[32])
	cluesMaster, _ := parseMinigameHiscore("Clue Scrolls - Master", allScores[33])
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
	url := fmt.Sprintf(
		"https://secure.runescape.com/m=hiscore_oldschool%s/index_lite.ws?player=%s",
		mode.urlComponent,
		player,
	)

	resp, err := http.Get(url)

	if err != nil {
		return Hiscores{}, err
	}

	defer resp.Body.Close()

	// Any other status code means the player does not exist in the given mode
	if resp.StatusCode != 200 {
		return Hiscores{}, &HiscoreAPIError{player, mode}
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	body := string(bodyBytes)
	return parseCSVHiscores(body), nil
}
