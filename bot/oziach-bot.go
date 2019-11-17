package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gempir/go-twitch-irc"
)

// OziachBot Wrapper class for twitch.Client that encompasses all OziachBot features
type OziachBot struct {
	TwitchClient   *twitch.Client
	dynamoDBClient *dynamodb.DynamoDB
}

// Channel DynamoDB schema for channel records
type Channel struct {
	Name string
}

// InitBot Initalizes OziachBot with callbacks and channels it needs to join
func InitBot() OziachBot {
	tClient := twitch.NewClient("OziachBot", fmt.Sprintf("oauth:%s", os.Getenv("OZIACH_AUTH")))

	// DynamoDB client connection
	credentialProvider := &credentials.EnvProvider{}
	credentials := credentials.NewCredentials(credentialProvider)
	dbConfig := aws.NewConfig().WithCredentials(credentials).WithRegion("us-west-1")
	session := session.New(dbConfig)
	dbClient := dynamodb.New(session)

	tableName := "ob-channels"
	scanInput := &dynamodb.ScanInput{
		TableName: &tableName,
	}

	result, err := dbClient.Scan(scanInput)
	if err != nil {
		panic(err)
	}

	bot := OziachBot{
		TwitchClient:   tClient,
		dynamoDBClient: dbClient,
	}

	tClient.OnNewMessage(bot.HandleMessage)

	for _, item := range result.Items {
		channel := Channel{}
		dynamodbattribute.UnmarshalMap(item, &channel)
		tClient.Join(channel.Name)
	}

	return bot
}

// Connect Connects OziachBot to Twitch IRC
func (bot *OziachBot) Connect() error {
	return bot.TwitchClient.Connect()
}

// HandleMessage Main callback method to wrap all actions on a PRIVMSG
func (bot *OziachBot) HandleMessage(channel string, user twitch.User, message twitch.Message) {
	idx := strings.Index(message.Text, " ")
	if idx == -1 {
		idx = len(message.Text)
	}

	switch message.Text[:idx] {
	case "!lvl", "!level":
		tokens := strings.SplitN(message.Text, " ", 3)
		if len(tokens) < 3 {
			break
		}

		skillName := tokens[1]
		// Truncate player to 12 characters, max length of an OSRS username
		player := tokens[2][:12]

		go bot.HandleSkillLookup(channel, user, skillName, player)
	case "!total", "!overall":
		tokens := strings.SplitN(message.Text, " ", 2)
		if len(tokens) < 2 {
			break
		}

		skillName := "Overall"
		// Truncate player to 12 characters, max length of an OSRS username
		player := tokens[1][:12]

		go bot.HandleSkillLookup(channel, user, skillName, player)
	}
}

// Say Wrapper for Client.Say that prefixes the text with "/me"
func (bot *OziachBot) Say(channel, text string) {
	formattedText := fmt.Sprintf("/me %s", text)
	bot.TwitchClient.Say(channel, formattedText)
}
