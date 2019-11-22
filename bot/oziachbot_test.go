package bot

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
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
			t.Error("Channel not joined")
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

	t.Run("NewChannel", func(t *testing.T) {
		name := "new channel"
		go bot.ConnectToChannel(name)

		select {
		case j := <-bot.ChannelDB.(*mockChannelDB).addChan:
			if j.Name != name {
				t.Errorf("Saved %s, but expected to save %s", j.Name, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("New channel not saved")
		}

		select {
		case j := <-bot.TwitchClient.(*mockIRC).joinChan:
			if j != name {
				t.Errorf("Joined %s, but expected to join %s", j, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("New channel not joined")
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
			t.Error("Existing channel not updated")
		}

		select {
		case j := <-bot.TwitchClient.(*mockIRC).joinChan:
			if j != name {
				t.Errorf("Joined %s, but expected to join %s", j, name)
			}
		case <-time.After(3 * time.Second):
			t.Error("New channel not joined")
		}
	})
}
