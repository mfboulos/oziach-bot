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
	Rank  int
	Level int
	Exp   int
}

// MinigameHiscore struct representing the hiscore of anything that's not a skill
type MinigameHiscore struct {
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
	skills []SkillHiscore

	bhHunter MinigameHiscore
	bhRogue  MinigameHiscore

	lms MinigameHiscore

	clues []MinigameHiscore
}

// UnrankedError Returned when a hiscore doesn't exist (player is unranked)
type UnrankedError struct{}

// HiscoreAPIError Returned when a player's hiscores aren't available for the given GameMode
type HiscoreAPIError struct {
	Player string
	Mode   GameMode
}

type hiscoreAPI interface {
	GetHiscoresAPIResponse(player string, mode GameMode) (string, error)
}

type hiscoreAPIImpl struct{}

func (hiscoreAPIImpl) GetHiscoresAPIResponse(player string, mode GameMode) (string, error) {
	url := fmt.Sprintf(
		"https://secure.runescape.com/m=hiscore_oldschool%s/index_lite.ws?player=%s",
		mode.urlComponent,
		player,
	)

	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Any other status code means the player does not exist in the given mode
	if resp.StatusCode != 200 {
		return "", &HiscoreAPIError{player, mode}
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	return string(bodyBytes), nil
}

var (
	// Enumerated GameMode values
	normal          GameMode = GameMode{"Normal", ""}
	ironman         GameMode = GameMode{"Ironman", "_ironman"}
	hardcoreIronman GameMode = GameMode{"Hardcore Ironman", "_hardcore_ironman"}
	ultimateIronman GameMode = GameMode{"Ultimate Ironman", "_ultimate"}

	// Skill names concurrent to Hiscores.skills
	skillNames []string = []string{
		"Overall",
		"Attack",
		"Defense",
		"Strength",
		"Hitpoints",
		"Ranged",
		"Prayer",
		"Magic",
		"Cooking",
		"Woodcutting",
		"Fletching",
		"Fishing",
		"Firemaking",
		"Crafting",
		"Smithing",
		"Mining",
		"Herblore",
		"Agility",
		"Thieving",
		"Slayer",
		"Farming",
		"Runecraft",
		"Hunter",
		"Construction",
	}

	// Clue score names concurrent to Hiscores.clues
	clueNames []string = []string{
		"Overall",
		"Beginner",
		"Easy",
		"Medium",
		"Hard",
		"Elite",
		"Master",
	}
)

// Skill Enum value for skill
type Skill int

// Clue Enum value for clue type
type Clue int

// Enumerated values for index retrieval of array-based scores
const (
	Overall      Skill = 0
	Attack       Skill = 1
	Defense      Skill = 2
	Strength     Skill = 3
	Hitpoints    Skill = 4
	Ranged       Skill = 5
	Prayer       Skill = 6
	Magic        Skill = 7
	Cooking      Skill = 8
	Woodcutting  Skill = 9
	Fletching    Skill = 10
	Fishing      Skill = 11
	Firemaking   Skill = 12
	Crafting     Skill = 13
	Smithing     Skill = 14
	Mining       Skill = 15
	Herblore     Skill = 16
	Agility      Skill = 17
	Thieving     Skill = 18
	Slayer       Skill = 19
	Farming      Skill = 20
	Runecraft    Skill = 21
	Hunter       Skill = 22
	Construction Skill = 23

	OverallClues  Clue = 0
	BeginnerClues Clue = 1
	EasyClues     Clue = 2
	MediumClues   Clue = 3
	HardClues     Clue = 4
	EliteClues    Clue = 5
	MasterClues   Clue = 6
)

func (e *UnrankedError) Error() string {
	return "Player is not ranked in this skill/minigame"
}

func (e *HiscoreAPIError) Error() string {
	return fmt.Sprintf("%s is not a(n) %s account", e.Player, e.Mode.Name)
}

// SameScores Returns true if the hiscores represent the same account, false otherwise
func SameScores(h1, h2 Hiscores) bool {
	for i, skill := range h1.skills {
		if skill.Exp != h2.skills[i].Exp {
			return false
		}
	}

	return true
}

// LookupHiscores Retrieves the hiscore based on the most restrictive
// GameMode pertaining to the player
//
// For example, a Hardcore Ironman has a Normal, Ironman, and Hardcore Ironman
// hiscore entry, but here we only return the Hardcore Ironman entry
func LookupHiscores(player string) (Hiscores, GameMode, error) {
	return lookupHiscores(player, hiscoreAPIImpl{})
}

func lookupHiscores(player string, builder hiscoreAPI) (Hiscores, GameMode, error) {
	mode := normal
	playerHiscores, err := lookupHiscoresByGameMode(player, mode, builder)

	// All players should be in normal hiscores. If they are not, they are either unranked
	// or they do not exist
	if err != nil {
		return playerHiscores, mode, err
	}

	ironmanHiscores, err := lookupHiscoresByGameMode(player, ironman, builder)
	if err == nil {
		hardcoreHiscores, err := lookupHiscoresByGameMode(player, hardcoreIronman, builder)
		ultimateHiscores, err2 := lookupHiscoresByGameMode(player, ultimateIronman, builder)

		// To determine what kind of ironman we have, first we check to see if there is a
		// hiscore under that GameMode. If experience values in that GameMode and
		// ironman mode match, the player is in that GameMode
		if err == nil && SameScores(hardcoreHiscores, ironmanHiscores) {
			playerHiscores = hardcoreHiscores
			mode = hardcoreIronman
		} else if err2 == nil && SameScores(ultimateHiscores, ironmanHiscores) {
			playerHiscores = ultimateHiscores
			mode = ultimateIronman
		} else {
			playerHiscores = ironmanHiscores
			mode = ironman
		}
	}

	return playerHiscores, mode, nil
}

// GetSkillHiscoreFromName maps string name to a specific hiscore, returns that score
// with its official name
func (hiscores Hiscores) GetSkillHiscoreFromName(name string) (string, SkillHiscore, error) {
	// Skill name and alias mapping to individual skill hiscores
	skillMap := map[string]Skill{
		"overall":      Overall,
		"total":        Overall,
		"attack":       Attack,
		"atk":          Attack,
		"defense":      Defense,
		"def":          Defense,
		"strength":     Strength,
		"str":          Strength,
		"hitpoints":    Hitpoints,
		"hp":           Hitpoints,
		"ranged":       Ranged,
		"range":        Ranged,
		"ranging":      Ranged,
		"prayer":       Prayer,
		"pray":         Prayer,
		"magic":        Magic,
		"mage":         Magic,
		"magician":     Magic,
		"cooking":      Cooking,
		"cook":         Cooking,
		"woodcutting":  Woodcutting,
		"woodcut":      Woodcutting,
		"wc":           Woodcutting,
		"fletching":    Fletching,
		"fletch":       Fletching,
		"fishing":      Fishing,
		"fish":         Fishing,
		"firemaking":   Firemaking,
		"fm":           Firemaking,
		"crafting":     Crafting,
		"craft":        Crafting,
		"smithing":     Smithing,
		"smith":        Smithing,
		"mining":       Mining,
		"mine":         Mining,
		"herblore":     Herblore,
		"herb":         Herblore,
		"agility":      Agility,
		"agil":         Agility,
		"thieving":     Thieving,
		"thieve":       Thieving,
		"thiev":        Thieving,
		"slayer":       Slayer,
		"slay":         Slayer,
		"farming":      Farming,
		"farm":         Farming,
		"kkona":        Farming,
		"runecraft":    Runecraft,
		"rc":           Runecraft,
		"hunter":       Hunter,
		"hunting":      Hunter,
		"hunt":         Hunter,
		"construction": Construction,
		"con":          Construction,
	}

	if skill, ok := skillMap[strings.ToLower(name)]; ok {
		return skillNames[skill], hiscores.skills[skill], nil
	}

	return "", SkillHiscore{}, errors.New("Could not map name to skill")
}

func parseSkillHiscore(hiscore string) (SkillHiscore, error) {
	skill := SkillHiscore{}
	vals := strings.Split(hiscore, ",")
	skill.Rank, _ = strconv.Atoi(vals[0])
	skill.Level, _ = strconv.Atoi(vals[1])
	skill.Exp, _ = strconv.Atoi(vals[2])
	return skill, nil
}

func parseMinigameHiscore(hiscore string) (MinigameHiscore, error) {
	minigame := MinigameHiscore{}

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
	hiscores.skills = make([]SkillHiscore, 24)
	for i, skill := range allScores[:24] {
		hiscores.skills[i], _ = parseSkillHiscore(skill)
	}

	// Bounty Hunter mappings
	hiscores.bhHunter, _ = parseMinigameHiscore(allScores[24])
	hiscores.bhRogue, _ = parseMinigameHiscore(allScores[25])

	// LMS mapping
	hiscores.lms, _ = parseMinigameHiscore(allScores[26])

	// Clue mappings
	hiscores.clues = make([]MinigameHiscore, 7)
	for i, clue := range allScores[27:34] {
		hiscores.clues[i], _ = parseMinigameHiscore(clue)
	}

	return hiscores
}

// LookupHiscoresByGameMode Looks up a player's hiscores ranked according to the given GameMode
func LookupHiscoresByGameMode(player string, mode GameMode) (Hiscores, error) {
	return lookupHiscoresByGameMode(player, mode, hiscoreAPIImpl{})
}

func lookupHiscoresByGameMode(player string, mode GameMode, builder hiscoreAPI) (Hiscores, error) {
	csv, err := builder.GetHiscoresAPIResponse(player, mode)

	if err != nil {
		return Hiscores{}, err
	}

	return parseCSVHiscores(csv), nil
}
