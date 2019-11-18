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
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/gempir/go-twitch-irc"
)

var (
	// TableName Name of the table holding Channel records in DynamoDB
	TableName string = "ob-channels"
)

// OziachBot Object structure containing all necessary clients and connections
// for OziachBot
type OziachBot struct {
	TwitchClient *twitch.Client
	ChannelDB    *dynamodb.DynamoDB
}

// Channel DynamoDB schema for channel records
type Channel struct {
	Name        string
	IsConnected bool
}

// UnmarshalChannel Convenience method to unmarshal a DynamoDB record directly
// into a Channel object
func UnmarshalChannel(item map[string]*dynamodb.AttributeValue) (Channel, error) {
	channel := Channel{}
	err := dynamodbattribute.UnmarshalMap(item, &channel)
	return channel, err
}

// UpdateChannel Updates an existing Channel record in the DB based on the given Expression
func (bot *OziachBot) UpdateChannel(name string, expr expression.Expression) (Channel, error) {
	// Anonymous struct with just the channel name
	channelKey := struct {
		Name string
	}{Name: name}

	marshalledKey, err := dynamodbattribute.MarshalMap(channelKey)

	if err != nil {
		return Channel{}, err
	}

	existsExpression, err := expression.NewBuilder().WithCondition(
		expression.Name("Name").AttributeExists(),
	).Build()

	if err != nil {
		return Channel{}, err
	}

	updateItemInput := &dynamodb.UpdateItemInput{}
	updateItemInput.SetTableName(TableName)
	updateItemInput.SetConditionExpression(*existsExpression.Condition())
	updateItemInput.SetReturnValues(dynamodb.ReturnValueAllNew)
	updateItemInput.SetKey(marshalledKey)
	updateItemInput.SetUpdateExpression(*expr.Update())
	output, err := bot.ChannelDB.UpdateItem(updateItemInput)

	if err != nil {
		return Channel{}, err
	}

	return UnmarshalChannel(output.Attributes)
}

// AddChannel Adds a new channel by name to the channel DB. Fails if a record
// with that Name already exists
func (bot *OziachBot) AddChannel(name string) (Channel, error) {
	channel := Channel{Name: name}
	marshalledChannel, err := dynamodbattribute.MarshalMap(channel)

	if err != nil {
		return channel, err
	}

	expression, err := expression.NewBuilder().WithCondition(
		expression.Name("Name").AttributeNotExists(),
	).Build()

	if err != nil {
		return channel, err
	}

	putItemInput := &dynamodb.PutItemInput{}
	putItemInput.SetTableName(TableName)
	putItemInput.SetItem(marshalledChannel)
	putItemInput.SetConditionExpression(*expression.Condition())
	_, err = bot.ChannelDB.PutItem(putItemInput)

	return channel, err
}

// GetChannel Gets the channel record by primary ID (Name)
func (bot *OziachBot) GetChannel(name string) (Channel, error) {
	// Anonymous struct with just the channel name
	channelKey := struct {
		Name string
	}{Name: name}
	marshalledKey, err := dynamodbattribute.MarshalMap(channelKey)

	if err != nil {
		return Channel{}, err
	}

	getItemInput := &dynamodb.GetItemInput{}
	getItemInput.SetTableName(TableName)
	getItemInput.SetKey(marshalledKey)
	output, err := bot.ChannelDB.GetItem(getItemInput)

	if err != nil {
		return Channel{}, err
	}

	return UnmarshalChannel(output.Item)
}

// DisconnectFromChannel Updates the record corresponding to the named channel by
// setting IsConnected to false, then OziachBot departs from the channel. Does not
// delete the DB record
func (bot *OziachBot) DisconnectFromChannel(name string) error {
	// Expression to set IsConnected to false
	expression, err := expression.NewBuilder().WithUpdate(
		expression.Set(expression.Name("IsConnected"), expression.Value(false)),
	).Build()

	if err != nil {
		return err
	}

	channel, err := bot.UpdateChannel(name, expression)

	if err == nil {
		bot.TwitchClient.Depart(channel.Name)
	}

	return err
}

// ConnectToChannel Adds a new channel record to the DB if it does not yet exist.
// Otherwise, updates the existing channel by setting IsConnected to true. Then
// OziachBot joins the channel
func (bot *OziachBot) ConnectToChannel(name string) error {
	channel, err := bot.AddChannel(name)

	if err != nil {
		// Expression to set IsConnected to false
		expression, err := expression.NewBuilder().WithUpdate(
			expression.Set(expression.Name("IsConnected"), expression.Value(true)),
		).Build()

		if err != nil {
			return err
		}

		channel, err = bot.UpdateChannel(name, expression)
	}

	if err == nil {
		bot.TwitchClient.Join(channel.Name)
	}

	return err
}

// InitBot Initalizes OziachBot with callbacks and channels it needs to join
func InitBot() OziachBot {
	// Twitch IRC client configuration
	twitchClient := twitch.NewClient(
		"OziachBot",
		fmt.Sprintf("oauth:%s", os.Getenv("OZIACH_AUTH")),
	)

	// DynamoDB client connection
	credentialProvider := &credentials.EnvProvider{}
	credentials := credentials.NewCredentials(credentialProvider)
	dbConfig := aws.NewConfig().WithCredentials(credentials).WithRegion("us-west-1")
	session := session.New(dbConfig)
	dbClient := dynamodb.New(session)

	// Read all records from channel DB
	scanInput := &dynamodb.ScanInput{TableName: &TableName}
	result, err := dbClient.Scan(scanInput)
	if err != nil {
		panic(err)
	}

	// Join all rooms from the DB query
	for _, item := range result.Items {
		channel, err := UnmarshalChannel(item)
		if err != nil {
			panic(err)
		}
		twitchClient.Join(channel.Name)
	}

	bot := OziachBot{
		TwitchClient: twitchClient,
		ChannelDB:    dbClient,
	}

	bot.TwitchClient.OnNewMessage(bot.HandleMessage)
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
