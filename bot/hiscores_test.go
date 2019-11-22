package bot

import (
	"fmt"
	"strings"
	"testing"
)

type mockHiscoreAPIClient struct{}

func (mock mockHiscoreAPIClient) GetAPIResponse(player string, mode GameMode) (string, error) {
	mockScores := []string{
		"1140740,922,26362111", // Skills start here
		"1600556,50,102080",
		"-1,1,0",
		"396686,90,5595374",
		"505273,86,3732405",
		"342695,90,5866885",
		"1170583,45,61571",
		"157706,96,10156589",
		"1721877,41,43025",
		"-1,36,26525",
		"1486638,27,10509",
		"-1,18,3597",
		"1101979,50,102330",
		"1946033,28,11634",
		"1875902,30,13743",
		"1997814,32,16889",
		"1315071,17,3195",
		"1544271,32,18238",
		"692214,53,138166",
		"1490078,23,6530",
		"1150078,10,1180",
		"-1,1,0",
		"-1,1,0",
		"329760,65,451646",
		"5106,661", // Bounty Hunter starts here
		"30275,36",
		"3308,992", // LMS
		"-1,-1",    // Clues start here
		"-1,-1",
		"-1,-1",
		"-1,-1",
		"-1,-1",
		"-1,-1",
		"-1,-1",
	}

	if player == fallenHardcoreAccount && mode != GameModeHardcoreIronman {
		mockScores[0] = mockScores[0] + "9"
	}

	var isValid bool

	switch mode {
	case GameModeUltimateIronman:
		isValid = false
	case GameModeHardcoreIronman:
		isValid = player == hardcoreAccount || player == fallenHardcoreAccount
	case GameModeIronman:
		isValid = player != notAnAccount && player != normalAccount
	case GameModeNormal:
		isValid = player != notAnAccount
	}

	if isValid {
		return strings.Join(mockScores, " "), nil
	}

	return "", &HiscoreAPIError{player, mode}
}

const (
	normalAccount         string = "Normal"
	ironmanAccount        string = "Ironman"
	hardcoreAccount       string = "HCIM"
	fallenHardcoreAccount string = "Fallen HCIM"
	notAnAccount          string = "Invalid Acc"
)

func NewMockHiscoreAPI() *HiscoreAPI {
	return &HiscoreAPI{
		Client: &mockHiscoreAPIClient{},
	}
}

func TestNewOSRSHiscoreAPI(t *testing.T) {
	api := NewOSRSHiscoreAPI()

	// We consider the API object to be valid if it uses the Hiscore API
	// client provided by the hiscores package
	if _, ok := api.Client.(*OSRSHiscoreAPIClient); !ok {
		t.Fatalf("New OSRS Hiscore API does not use OSRS Hiscore API Client")
	}
}

func TestFormatHiscoreAPIURL(t *testing.T) {
	type urlFormatTestCase struct {
		player   string
		mode     GameMode
		expected string
	}

	testCases := []urlFormatTestCase{
		urlFormatTestCase{
			player: normalAccount,
			mode:   GameModeUltimateIronman,
			expected: fmt.Sprintf(
				"https://secure.runescape.com/m=hiscore_oldschool_ultimate/index_lite.ws?player=%s",
				normalAccount,
			),
		},
		urlFormatTestCase{
			player: ironmanAccount,
			mode:   GameModeNormal,
			expected: fmt.Sprintf(
				"https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player=%s",
				ironmanAccount,
			),
		},
		urlFormatTestCase{
			player: notAnAccount,
			mode:   GameModeHardcoreIronman,
			expected: fmt.Sprintf(
				"https://secure.runescape.com/m=hiscore_oldschool_hardcore_ironman/index_lite.ws?player=%s",
				notAnAccount,
			),
		},
	}

	for _, tc := range testCases {
		actual := FormatHiscoreAPIURL(tc.player, tc.mode)
		if actual != tc.expected {
			t.Errorf("Hiscore API URL was %s, expected %s",
				actual,
				tc.expected,
			)
		}
	}
}

func TestHiscoresLookup(t *testing.T) {
	mockAPI := NewMockHiscoreAPI()

	t.Run("ByMode", func(t *testing.T) {
		type hiscoreTestCase struct {
			player     string
			mode       GameMode
			shouldPass bool
		}

		testCaseMap := map[string][]hiscoreTestCase{
			"SameMode": []hiscoreTestCase{
				{normalAccount, GameModeNormal, true},
				{ironmanAccount, GameModeIronman, true},
				{hardcoreAccount, GameModeHardcoreIronman, true},
			},
			"LowerMode": []hiscoreTestCase{
				{hardcoreAccount, GameModeNormal, true},
				{hardcoreAccount, GameModeIronman, true},
			},
			"InvalidPlayer": []hiscoreTestCase{
				{notAnAccount, GameModeNormal, false},
			},
			"IncompatibleMode": []hiscoreTestCase{
				{hardcoreAccount, GameModeUltimateIronman, false},
				{normalAccount, GameModeIronman, false},
			},
		}

		for name, testCases := range testCaseMap {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				for _, tc := range testCases {
					t.Run(fmt.Sprintf("%s in %s Mode", tc.player, tc.mode.Name), func(t *testing.T) {
						t.Parallel()
						_, err := mockAPI.LookupHiscoresByGameMode(tc.player, tc.mode)

						if err == nil && !tc.shouldPass {
							t.Fatalf("Expected lookup failure, got success")
						}

						if err != nil && tc.shouldPass {
							t.Fatalf("Expected lookup success, got failure")
						}
					})
				}
			})
		}
	})

	t.Run("ModeAgnostic", func(t *testing.T) {
		t.Run("InvalidPlayer", func(t *testing.T) {
			t.Parallel()
			player := notAnAccount

			_, _, err := mockAPI.LookupHiscores(player)

			if err == nil {
				t.Fatalf("Hiscore lookup succeeded for invalid player")
			}
		})

		players := map[string]GameMode{
			normalAccount:         GameModeNormal,
			ironmanAccount:        GameModeIronman,
			hardcoreAccount:       GameModeHardcoreIronman,
			fallenHardcoreAccount: GameModeIronman,
		}

		for player, expectedMode := range players {
			t.Run(player, func(t *testing.T) {
				_, actualMode, err := mockAPI.LookupHiscores(player)

				if err != nil {
					t.Fatalf("Hiscore lookup failed for valid player")
				}

				if expectedMode != actualMode {
					t.Errorf("Expected %s, got %s", expectedMode.Name, actualMode.Name)
				}
			})
		}
	})
}

func TestGetSkillHiscoreFromName(t *testing.T) {
	player := normalAccount
	mode := GameModeNormal
	mockAPI := NewMockHiscoreAPI()
	hiscores, err := mockAPI.LookupHiscoresByGameMode(player, mode)

	if err != nil {
		t.Errorf("Hiscore lookup failed for valid player")
	}

	t.Run("ExactName", func(t *testing.T) {
		t.Parallel()

		name := "Ranged"
		expected := "Ranged"
		skillName, _, err := hiscores.GetSkillHiscoreFromName(name)

		if err != nil {
			t.Fatal(err)
		}

		if skillName != expected {
			t.Errorf("Incorrect skill retrieved: expected %s, got %s", expected, skillName)
		}
	})
	t.Run("Alias", func(t *testing.T) {
		t.Parallel()

		name := "wc"
		expected := "Woodcutting"
		skillName, _, err := hiscores.GetSkillHiscoreFromName(name)

		if err != nil {
			t.Fatal(err)
		}

		if skillName != expected {
			t.Errorf("Incorrect skill retrieved: expected %s, got %s", expected, skillName)
		}
	})
	t.Run("WrongName", func(t *testing.T) {
		t.Parallel()

		name := "Sailing"
		_, _, err := hiscores.GetSkillHiscoreFromName(name)

		if err == nil {
			t.Errorf("Skill hiscore lookup succeeded with invalid skill")
		}
	})
}
