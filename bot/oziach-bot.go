package bot

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// ChannelNotFoundError Returned when an operation requires an existing channel
// that isn't found
type ChannelNotFoundError struct {
	Channel string
}

func (e ChannelNotFoundError) Error() string {
	return fmt.Sprintf("Channel %s not found", e.Channel)
}

// ChannelAlreadyExistsError Returned when an operation requires a channel to
// be new, but it already exists
type ChannelAlreadyExistsError struct {
	Channel string
}

func (e ChannelAlreadyExistsError) Error() string {
	return fmt.Sprintf("Channel %s already exists", e.Channel)
}

// UpdateChannel Updates an existing Channel record in the DB. Builds an expression
// based on the given builder by adding attribute existence check on the primary key.
// Returns ChannelNotFoundError if the existence check fails
func (bot *OziachBot) UpdateChannel(name string, builder expression.Builder) (Channel, error) {
	// Anonymous struct with just the channel name
	channelKey := struct {
		Name string
	}{Name: name}

	marshalledKey, err := dynamodbattribute.MarshalMap(channelKey)

	if err != nil {
		return Channel{}, err
	}

	expression, err := builder.WithCondition(
		expression.Name("Name").AttributeExists(),
	).Build()

	if err != nil {
		return Channel{}, err
	}

	updateItemInput := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  expression.Names(),
		ExpressionAttributeValues: expression.Values(),
		ConditionExpression:       expression.Condition(),
		UpdateExpression:          expression.Update(),
	}
	updateItemInput.SetTableName(TableName)
	updateItemInput.SetReturnValues(dynamodb.ReturnValueAllNew)
	updateItemInput.SetKey(marshalledKey)
	output, err := bot.ChannelDB.UpdateItem(updateItemInput)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				err = ChannelNotFoundError{name}
			}
		}

		return Channel{}, err
	}

	return UnmarshalChannel(output.Attributes)
}

// AddChannel Adds a new channel by name to the channel DB. Fails if a record
// with that Name already exists
func (bot *OziachBot) AddChannel(name string) (Channel, error) {
	channel := Channel{
		Name:        name,
		IsConnected: true,
	}
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

	putItemInput := &dynamodb.PutItemInput{
		ExpressionAttributeNames:  expression.Names(),
		ExpressionAttributeValues: expression.Values(),
		ConditionExpression:       expression.Condition(),
	}
	putItemInput.SetTableName(TableName)
	putItemInput.SetItem(marshalledChannel)
	_, err = bot.ChannelDB.PutItem(putItemInput)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				err = ChannelAlreadyExistsError{name}
			}
		}
	}

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

	// GetItem doesn't set Item if it doesn't find anything, so
	// we check for a "zero map"
	if len(output.Item) == 0 {
		return Channel{}, ChannelNotFoundError{name}
	}

	return UnmarshalChannel(output.Item)
}

// DisconnectFromChannel Updates the record corresponding to the named channel by
// setting IsConnected to false, then OziachBot departs from the channel. Does not
// delete the DB record
func (bot *OziachBot) DisconnectFromChannel(name string) error {
	// Expression builder to set IsConnected to false
	builder := expression.NewBuilder().WithUpdate(
		expression.Set(expression.Name("IsConnected"), expression.Value(false)),
	)

	log.Println("Attempting to disconnect from", name)
	channel, err := bot.UpdateChannel(name, builder)

	if err == nil {
		bot.TwitchClient.Depart(channel.Name)
		log.Println("Disconnect successful")
	} else {
		log.Println("Disconnect failed")
	}

	return err
}

// ConnectToChannel Adds a new channel record to the DB if it does not yet exist.
// Otherwise, updates the existing channel by setting IsConnected to true. Then
// OziachBot joins the channel
func (bot *OziachBot) ConnectToChannel(name string) error {
	log.Println("Attempting to connect to", name)
	channel, err := bot.AddChannel(name)

	if err != nil {
		// Expression builder to set IsConnected to true
		builder := expression.NewBuilder().WithUpdate(
			expression.Set(expression.Name("IsConnected"), expression.Value(true)),
		)

		channel, err = bot.UpdateChannel(name, builder)
	}

	if err == nil {
		bot.TwitchClient.Join(channel.Name)
		log.Println("Connection successful")
	} else {
		log.Println("Connection failed")
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

	log.Println("Connecting to DynamoDB")
	// DynamoDB client connection
	credentialProvider := &credentials.EnvProvider{}
	credentials := credentials.NewCredentials(credentialProvider)
	dbConfig := aws.NewConfig().WithCredentials(credentials).WithRegion("us-west-1")
	session := session.New(dbConfig)
	dbClient := dynamodb.New(session)

	log.Println("Reading channels from DB")
	// Read all records from channel DB
	scanInput := &dynamodb.ScanInput{TableName: &TableName}
	result, err := dbClient.Scan(scanInput)
	if err != nil {
		panic(err)
	}

	log.Println("Joining channels")
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
	go bot.ServeAPI()
	return bot
}

// Connect Connects OziachBot to Twitch IRC
func (bot *OziachBot) Connect() error {
	log.Println("Connecting to Twitch IRC")
	return bot.TwitchClient.Connect()
}

// HandleMessage Main callback method to wrap all actions on a PRIVMSG
func (bot *OziachBot) HandleMessage(channel string, user twitch.User, message twitch.Message) {
	log.Printf("Handing message \"%s\" from channel %s\n", message.Text, channel)
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
