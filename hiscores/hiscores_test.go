package hiscores

import (
	"fmt"
	"strings"
	"testing"
)

type mockHiscoreAPI struct{}

func (mock mockHiscoreAPI) GetHiscoresAPIResponse(player string, mode GameMode) (string, error) {
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

	if player == fallenHardcoreAccount && mode != hardcoreIronman {
		mockScores[0] = mockScores[0] + "9"
	}

	var isValid bool

	switch mode {
	case ultimateIronman:
		isValid = false
	case hardcoreIronman:
		isValid = player == hardcoreAccount || player == fallenHardcoreAccount
	case ironman:
		isValid = player != notAnAccount && player != normalAccount
	case normal:
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
	hardcoreAccount       string = "Harcore Ironman"
	fallenHardcoreAccount string = "Fallen Hardcore Ironman"
	notAnAccount          string = "Invalid Account"
)

func TestHiscoresLookup(t *testing.T) {
	mockAPI := mockHiscoreAPI{}

	t.Run("ByMode", func(t *testing.T) {
		type hiscoreTestCase struct {
			player     string
			mode       GameMode
			shouldPass bool
		}

		testCaseMap := map[string][]hiscoreTestCase{
			"SameMode": []hiscoreTestCase{
				{normalAccount, normal, true},
				{ironmanAccount, ironman, true},
				{hardcoreAccount, hardcoreIronman, true},
			},
			"LowerMode": []hiscoreTestCase{
				{hardcoreAccount, normal, true},
				{hardcoreAccount, ironman, true},
			},
			"InvalidPlayer": []hiscoreTestCase{
				{notAnAccount, normal, false},
			},
			"IncompatibleMode": []hiscoreTestCase{
				{hardcoreAccount, ultimateIronman, false},
				{normalAccount, ironman, false},
			},
		}

		for name, testCases := range testCaseMap {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				for _, tc := range testCases {
					t.Run(fmt.Sprintf("%s in %s Mode", tc.player, tc.mode.Name), func(t *testing.T) {
						t.Parallel()
						_, err := lookupHiscoresByGameMode(tc.player, tc.mode, mockAPI)

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

			_, _, err := lookupHiscores(player, mockAPI)

			if err == nil {
				t.Fatalf("Hiscore lookup succeeded for invalid player")
			}
		})

		players := map[string]GameMode{
			normalAccount:         normal,
			ironmanAccount:        ironman,
			hardcoreAccount:       hardcoreIronman,
			fallenHardcoreAccount: ironman,
		}

		for player, expectedMode := range players {
			t.Run(player, func(t *testing.T) {
				_, actualMode, err := lookupHiscores(player, mockAPI)

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

func TestHiscoresLookupFailsWithInvalidPlayer(t *testing.T) {
	player := notAnAccount
	mockAPI := mockHiscoreAPI{}

	_, _, err := lookupHiscores(player, mockAPI)

	if err == nil {
		t.Errorf("Hiscore lookup succeeded for invalid player")
	}
}

func TestHiscoresLookupEvaluatesCorrectGameMode(t *testing.T) {
	players := map[string]GameMode{
		normalAccount:         normal,
		ironmanAccount:        ironman,
		hardcoreAccount:       hardcoreIronman,
		fallenHardcoreAccount: ironman,
	}
	mockAPI := mockHiscoreAPI{}

	for player, mode := range players {
		_, mode2, err := lookupHiscores(player, mockAPI)

		if err != nil {
			t.Errorf("Hiscore lookup failed for valid player")
		}

		if mode != mode2 {
			t.Errorf("Mismatched GameModes: expected %s, got %s", mode.Name, mode2.Name)
		}
	}
}

func TestGetSkillHiscoreFromName(t *testing.T) {
	player := normalAccount
	mode := normal
	mockAPI := mockHiscoreAPI{}
	hiscores, err := lookupHiscoresByGameMode(player, mode, mockAPI)

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
