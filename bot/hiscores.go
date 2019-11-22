package bot

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var (
	// GameModeNormal Normal game mode
	GameModeNormal GameMode = GameMode{"Normal", ""}
	// GameModeIronman Ironman game mode
	GameModeIronman GameMode = GameMode{"Ironman", "_ironman"}
	// GameModeHardcoreIronman Hardcore Ironman game mode
	GameModeHardcoreIronman GameMode = GameMode{"Hardcore Ironman", "_hardcore_ironman"}
	// GameModeUltimateIronman Ultimate Ironman game mode
	GameModeUltimateIronman GameMode = GameMode{"Ultimate Ironman", "_ultimate"}

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
	SkillOverall      Skill = 0
	SkillAttack       Skill = 1
	SkillDefense      Skill = 2
	SkillStrength     Skill = 3
	SkillHitpoints    Skill = 4
	SkillRanged       Skill = 5
	SkillPrayer       Skill = 6
	SkillMagic        Skill = 7
	SkillCooking      Skill = 8
	SkillWoodcutting  Skill = 9
	SkillFletching    Skill = 10
	SkillFishing      Skill = 11
	SkillFiremaking   Skill = 12
	SkillCrafting     Skill = 13
	SkillSmithing     Skill = 14
	SkillMining       Skill = 15
	SkillHerblore     Skill = 16
	SkillAgility      Skill = 17
	SkillThieving     Skill = 18
	SkillSlayer       Skill = 19
	SkillFarming      Skill = 20
	SkillRunecraft    Skill = 21
	SkillHunter       Skill = 22
	SkillConstruction Skill = 23

	ClueOverallClues  Clue = 0
	ClueBeginnerClues Clue = 1
	ClueEasyClues     Clue = 2
	ClueMediumClues   Clue = 3
	ClueHardClues     Clue = 4
	ClueEliteClues    Clue = 5
	ClueMasterClues   Clue = 6
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

func (e *UnrankedError) Error() string {
	return "Player is not ranked in this skill/minigame"
}

func (e *HiscoreAPIError) Error() string {
	return fmt.Sprintf("%s is not a(n) %s account", e.Player, e.Mode.Name)
}

// HiscoreAPIClient Interface to abstract from any interactions with the Hiscore API
type HiscoreAPIClient interface {
	GetAPIResponse(player string, mode GameMode) (string, error)
}

// OSRSHiscoreAPIClient Implementation of HiscoreAPIClient that queries the OSRS Hiscore API
type OSRSHiscoreAPIClient struct{}

// HiscoreAPI Directs all features that interface with the Hiscore API
type HiscoreAPI struct {
	Client HiscoreAPIClient
}

// GetAPIResponse Sends a GET request to the OSRS Hiscore API according to the
// player and GameMode, and returns the response as a CSV string
func (osrsAPI *OSRSHiscoreAPIClient) GetAPIResponse(player string, mode GameMode) (string, error) {
	url := FormatHiscoreAPIURL(player, mode)

	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Any other status code means the player does not exist in the given mode
	if resp.StatusCode != 200 {
		return "", &HiscoreAPIError{player, mode}
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	return string(bodyBytes), err
}

// FormatHiscoreAPIURL Formats the base URL used to request hiscores from the OSRS Hiscore API
// based on the GameMode and adds the player as a query param
func FormatHiscoreAPIURL(player string, mode GameMode) string {
	return fmt.Sprintf(
		"https://secure.runescape.com/m=hiscore_oldschool%s/index_lite.ws?player=%s",
		mode.urlComponent,
		player,
	)
}

// NewOSRSHiscoreAPI Returns a Hiscore API with the OSRS API client implementation
func NewOSRSHiscoreAPI() *HiscoreAPI {
	return &HiscoreAPI{
		Client: &OSRSHiscoreAPIClient{},
	}
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
func (api *HiscoreAPI) LookupHiscores(player string) (Hiscores, GameMode, error) {
	mode := GameModeNormal
	playerHiscores, err := api.LookupHiscoresByGameMode(player, mode)

	// All players should be in normal hiscores. If they are not, they are either unranked
	// or they do not exist
	if err != nil {
		return playerHiscores, mode, err
	}

	ironmanHiscores, err := api.LookupHiscoresByGameMode(player, GameModeIronman)
	if err == nil {
		hardcoreHiscores, err := api.LookupHiscoresByGameMode(player, GameModeHardcoreIronman)
		ultimateHiscores, err2 := api.LookupHiscoresByGameMode(player, GameModeUltimateIronman)

		// To determine what kind of ironman we have, first we check to see if there is a
		// hiscore under that GameMode. If experience values in that GameMode and
		// ironman mode match, the player is in that GameMode
		if err == nil && SameScores(hardcoreHiscores, ironmanHiscores) {
			playerHiscores = hardcoreHiscores
			mode = GameModeHardcoreIronman
		} else if err2 == nil && SameScores(ultimateHiscores, ironmanHiscores) {
			playerHiscores = ultimateHiscores
			mode = GameModeUltimateIronman
		} else {
			playerHiscores = ironmanHiscores
			mode = GameModeIronman
		}
	}

	return playerHiscores, mode, nil
}

// GetSkillHiscoreFromName maps string name to a specific hiscore, returns that score
// with its official name
func (hiscores Hiscores) GetSkillHiscoreFromName(name string) (string, SkillHiscore, error) {
	// Skill name and alias mapping to individual skill hiscores
	skillMap := map[string]Skill{
		"overall":      SkillOverall,
		"total":        SkillOverall,
		"attack":       SkillAttack,
		"atk":          SkillAttack,
		"defense":      SkillDefense,
		"def":          SkillDefense,
		"strength":     SkillStrength,
		"str":          SkillStrength,
		"hitpoints":    SkillHitpoints,
		"hp":           SkillHitpoints,
		"ranged":       SkillRanged,
		"range":        SkillRanged,
		"ranging":      SkillRanged,
		"prayer":       SkillPrayer,
		"pray":         SkillPrayer,
		"magic":        SkillMagic,
		"mage":         SkillMagic,
		"magician":     SkillMagic,
		"cooking":      SkillCooking,
		"cook":         SkillCooking,
		"woodcutting":  SkillWoodcutting,
		"woodcut":      SkillWoodcutting,
		"wc":           SkillWoodcutting,
		"fletching":    SkillFletching,
		"fletch":       SkillFletching,
		"fishing":      SkillFishing,
		"fish":         SkillFishing,
		"firemaking":   SkillFiremaking,
		"fm":           SkillFiremaking,
		"crafting":     SkillCrafting,
		"craft":        SkillCrafting,
		"smithing":     SkillSmithing,
		"smith":        SkillSmithing,
		"mining":       SkillMining,
		"mine":         SkillMining,
		"herblore":     SkillHerblore,
		"herb":         SkillHerblore,
		"agility":      SkillAgility,
		"agil":         SkillAgility,
		"thieving":     SkillThieving,
		"thieve":       SkillThieving,
		"thiev":        SkillThieving,
		"slayer":       SkillSlayer,
		"slay":         SkillSlayer,
		"farming":      SkillFarming,
		"farm":         SkillFarming,
		"kkona":        SkillFarming,
		"runecraft":    SkillRunecraft,
		"rc":           SkillRunecraft,
		"hunter":       SkillHunter,
		"hunting":      SkillHunter,
		"hunt":         SkillHunter,
		"construction": SkillConstruction,
		"con":          SkillConstruction,
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
func (api *HiscoreAPI) LookupHiscoresByGameMode(player string, mode GameMode) (Hiscores, error) {
	csv, err := api.Client.GetAPIResponse(player, mode)

	if err != nil {
		return Hiscores{}, err
	}

	return parseCSVHiscores(csv), nil
}
