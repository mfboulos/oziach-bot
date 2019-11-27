package bot

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/gempir/go-twitch-irc"
)

var (
	connectedChannel Channel = Channel{
		Name:        "channel1",
		IsConnected: true,
	}
	disconnectedChannel Channel = Channel{
		Name:        "channel2",
		IsConnected: true,
	}
)

type mockChannelDB struct {
	addChan    chan Channel
	updateChan chan string
}

func (db *mockChannelDB) GetChannel(name string) (Channel, error) {
	switch name {
	case connectedChannel.Name:
		return connectedChannel, nil
	case disconnectedChannel.Name:
		return disconnectedChannel, nil
	default:
		return Channel{}, ChannelNotFoundError{name}
	}
}

func (db *mockChannelDB) GetAllChannels() ([]Channel, error) {
	return []Channel{
		connectedChannel,
		disconnectedChannel,
	}, nil
}

func (db *mockChannelDB) AddChannel(name string) (Channel, error) {
	switch name {
	case connectedChannel.Name:
		return connectedChannel, ChannelAlreadyExistsError{name}
	case disconnectedChannel.Name:
		return disconnectedChannel, ChannelAlreadyExistsError{name}
	default:
		channel := Channel{
			Name:        name,
			IsConnected: true,
		}
		db.addChan <- channel
		return channel, nil
	}
}

func (db *mockChannelDB) UpdateChannel(name string, builder expression.Builder) (Channel, error) {
	switch name {
	case connectedChannel.Name:
		db.updateChan <- name
		return connectedChannel, nil
	case disconnectedChannel.Name:
		db.updateChan <- name
		return disconnectedChannel, nil
	default:
		return Channel{}, ChannelNotFoundError{name}
	}
}

type mockIRC struct {
	messageChan chan string
	joinChan    chan string
	departChan  chan string
	connected   bool
}

func (irc *mockIRC) Say(channel, text string) {
	irc.messageChan <- text
}

func (irc *mockIRC) Whisper(username, text string) {
	// Empty for now
}

func (irc *mockIRC) Join(channel string) {
	irc.joinChan <- channel
}

func (irc *mockIRC) Depart(channel string) {
	irc.departChan <- channel
}

func (irc *mockIRC) Userlist(channel string) ([]string, error) {
	return []string{}, nil
}

func (irc *mockIRC) Connect() error {
	irc.connected = true
	return nil
}

func (irc *mockIRC) Disconnect() error {
	irc.connected = false
	return nil
}

func NewMockBot() OziachBot {
	return OziachBot{
		TwitchClient: &mockIRC{
			messageChan: make(chan string),
			joinChan:    make(chan string),
			departChan:  make(chan string),
		},
		ChannelDB: &mockChannelDB{
			addChan:    make(chan Channel),
			updateChan: make(chan string),
		},
		HiscoreAPI: NewMockHiscoreAPI(),
	}
}

func TestInitBot(t *testing.T) {
	bot := NewMockBot()
	channels, _ := bot.ChannelDB.GetAllChannels()

	go func() {
		defer close(bot.TwitchClient.(*mockIRC).joinChan)
		bot.InitBot()
	}()

	for i := 0; i < len(channels); i++ {
		select {
		case <-bot.TwitchClient.(*mockIRC).joinChan:
		case <-time.After(3 * time.Second):
			t.Error("Join unsuccessful due to timeout")
		}
	}

	select {
	case <-bot.TwitchClient.(*mockIRC).joinChan:
		t.Error("Joined an unlisted channel")
	default:
	}
}

func TestConnectToChannel(t *testing.T) {
	bot := NewMockBot()

	t.Run("InvalidChannel", func(t *testing.T) {
		name := "new channel"
		wait := make(chan struct{})
		go func() {
			bot.ConnectToChannel(name)
			wait <- struct{}{}
		}()

		select {
		case <-bot.TwitchClient.(*mockIRC).joinChan:
			t.Error("Joined an invalid channel")
		case <-time.After(3 * time.Second):
			t.Error("Join unsuccessful due to timeout")
		case <-wait:
		}
	})

	t.Run("ExistingChannel", func(t *testing.T) {
		name := disconnectedChannel.Name
		go bot.ConnectToChannel(name)

		select {
		case j := <-bot.ChannelDB.(*mockChannelDB).updateChan:
			if j != name {
				t.Errorf("Updated %s, but expected to update %s", j, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("Update unsuccessful due to timeout")
		}

		select {
		case j := <-bot.TwitchClient.(*mockIRC).joinChan:
			if j != name {
				t.Errorf("Joined %s, but expected to join %s", j, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("Join unsuccessful due to timeout")
		}
	})
}

func TestDisconnectFromChannel(t *testing.T) {
	bot := NewMockBot()

	t.Run("NewChannel", func(t *testing.T) {
		name := "new channel"
		wait := make(chan error)
		go func() {
			err := bot.DisconnectFromChannel(name)
			wait <- err
		}()

		select {
		case <-bot.ChannelDB.(*mockChannelDB).updateChan:
			t.Error("Incorrect update of nonexistent channel")
		case <-bot.TwitchClient.(*mockIRC).departChan:
			t.Error("Incorrect departure of nonexistent channel")
		case <-time.After(3 * time.Second):
			t.Error("Disconnect unsuccessful due to timeout")
		case err := <-wait:
			if err == nil {
				t.Fatal("Expected error")
			}
		}
	})

	t.Run("ExistingChannel", func(t *testing.T) {
		name := connectedChannel.Name
		go bot.DisconnectFromChannel(name)

		select {
		case j := <-bot.ChannelDB.(*mockChannelDB).updateChan:
			if j != name {
				t.Errorf("Updated %s, but expected to update %s", j, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("Disconnect unsuccessful due to timeout")
		}

		select {
		case j := <-bot.TwitchClient.(*mockIRC).departChan:
			if j != name {
				t.Errorf("Departed %s, but expected to depart %s", j, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("Disconnect unsuccessful due to timeout")
		}
	})
}

func TestHandleMessage(t *testing.T) {
	bot := NewMockBot()

	t.Run("LevelCommand", func(t *testing.T) {
		testUser := twitch.User{
			Username:    "testuser",
			DisplayName: "TestUser",
		}

		t.Run("ValidInvocation", func(t *testing.T) {
			testMessage := twitch.Message{
				Text: fmt.Sprintf("!lvl ranged %s", ironmanAccount),
			}

			expected := "/me " + FormatSkillLookupOutput(
				testUser.DisplayName,
				ironmanAccount,
				"Ranged",
				GameModeIronman,
				SkillHiscore{
					Rank:  342695,
					Level: 90,
					Exp:   5866885,
				},
			)

			go bot.HandleMessage("whatever channel doesn't matter", testUser, testMessage)

			select {
			case resp := <-bot.TwitchClient.(*mockIRC).messageChan:
				if resp != expected {
					t.Errorf("Said %s, but expected to say %s", resp, expected)
				}
			case <-time.After(3 * time.Second):
				t.Error("Message handling unsuccessful due to timeout")
			}
		})

		t.Run("InvalidPlayer", func(t *testing.T) {
			testMessage := twitch.Message{
				Text: fmt.Sprintf("!lvl ranged %s", notAnAccount),
			}

			expected := fmt.Sprintf(
				"/me @%s Could not find player %s",
				testUser.DisplayName,
				notAnAccount,
			)

			go bot.HandleMessage("whatever channel doesn't matter", testUser, testMessage)

			select {
			case resp := <-bot.TwitchClient.(*mockIRC).messageChan:
				if resp != expected {
					t.Errorf("Said %s, but expected to say %s", resp, expected)
				}
			case <-time.After(3 * time.Second):
				t.Error("Message handling unsuccessful due to timeout")
			}
		})

		t.Run("InvalidSkill", func(t *testing.T) {
			testMessage := twitch.Message{
				Text: fmt.Sprintf("!lvl sailing %s", hardcoreAccount),
			}

			wait := make(chan struct{})
			go func() {
				bot.HandleMessage("whatever channel doesn't matter", testUser, testMessage)
				wait <- struct{}{}
			}()

			select {
			case <-bot.TwitchClient.(*mockIRC).messageChan:
				t.Errorf("Bot responded, but should fail silently")
			case <-time.After(3 * time.Second):
				t.Error("Message handling unsuccessful due to timeout")
			case <-wait:
			}
		})
	})

	t.Run("TotalCommand", func(t *testing.T) {
		testUser := twitch.User{
			Username:    "testuser",
			DisplayName: "TestUser",
		}

		t.Run("ValidInvocation", func(t *testing.T) {
			testMessage := twitch.Message{
				Text: fmt.Sprintf("!total %s", ironmanAccount),
			}

			expected := "/me " + FormatSkillLookupOutput(
				testUser.DisplayName,
				ironmanAccount,
				"Overall",
				GameModeIronman,
				SkillHiscore{
					Rank:  1140740,
					Level: 922,
					Exp:   26362111,
				},
			)

			go bot.HandleMessage("whatever channel doesn't matter", testUser, testMessage)

			select {
			case resp := <-bot.TwitchClient.(*mockIRC).messageChan:
				if resp != expected {
					t.Errorf("Said %s, but expected to say %s", resp, expected)
				}
			case <-time.After(3 * time.Second):
				t.Error("Message handling unsuccessful due to timeout")
			}
		})

		t.Run("InvalidPlayer", func(t *testing.T) {
			testMessage := twitch.Message{
				Text: fmt.Sprintf("!total %s", notAnAccount),
			}

			expected := fmt.Sprintf(
				"/me @%s Could not find player %s",
				testUser.DisplayName,
				notAnAccount,
			)

			go bot.HandleMessage("whatever channel doesn't matter", testUser, testMessage)

			select {
			case resp := <-bot.TwitchClient.(*mockIRC).messageChan:
				if resp != expected {
					t.Errorf("Said %s, but expected to say %s", resp, expected)
				}
			case <-time.After(3 * time.Second):
				t.Error("Message handling unsuccessful due to timeout")
			}
		})
	})
}
