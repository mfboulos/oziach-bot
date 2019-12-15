package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/gempir/go-twitch-irc"
)

var (
	// TableName Name of the table holding Channel records in DynamoDB
	TableName string = "ob-channels"

	// Usernames that are ignored when handling messages, stored as a set
	ignored map[string]struct{} = map[string]struct{}{
		"streamelements": struct{}{},
	}
)

// OziachBot Object structure containing all necessary clients and connections
// for OziachBot
type OziachBot struct {
	TwitchClient IRC
	ChannelDB    ChannelDatabase
	HiscoreAPI   *HiscoreAPI
}

// IRC Interface for interaction with an IRC Server
type IRC interface {
	Say(channel, text string)
	Whisper(username, text string)
	Join(channel string)
	Depart(channel string)
	Userlist(channel string) ([]string, error)
	Connect() error
	Disconnect() error
}

// ChannelDatabase Interface for CRUD operations on Channel database
type ChannelDatabase interface {
	GetChannel(name string) (Channel, error)
	GetAllChannels() ([]Channel, error)
	AddChannel(name string) (Channel, error)
	UpdateChannel(name string, builder expression.Builder) (Channel, error)
}

// DynamoDBChannelDatabase Implementation of ChannelDatabase that uses a
// DynamoDB client to access the database
type DynamoDBChannelDatabase struct {
	Client *dynamodb.DynamoDB
}

// Channel DynamoDB schema for channel records
type Channel struct {
	Name        string `json:"name"`
	IsConnected bool   `json:"isConnected"`
	RSN         string `json:"rsn"`
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

// GetChannel Gets the channel record by primary ID (Name)
func (db *DynamoDBChannelDatabase) GetChannel(name string) (Channel, error) {
	// Anonymous struct with just the channel name
	channelKey := struct {
		Name string `json:"name"`
	}{Name: name}
	marshalledKey, err := dynamodbattribute.MarshalMap(channelKey)

	if err != nil {
		return Channel{}, err
	}

	getItemInput := &dynamodb.GetItemInput{}
	getItemInput.SetTableName(TableName)
	getItemInput.SetKey(marshalledKey)
	output, err := db.Client.GetItem(getItemInput)

	// GetItem doesn't set Item if it doesn't find anything, so
	// we check for a "zero map"
	if len(output.Item) == 0 {
		return Channel{}, ChannelNotFoundError{name}
	}

	return UnmarshalChannel(output.Item)
}

// GetAllChannels Gets all channels from the database
func (db *DynamoDBChannelDatabase) GetAllChannels() ([]Channel, error) {
	// Scan all records from channel DB
	scanInput := &dynamodb.ScanInput{
		TableName: &TableName,
	}

	result, err := db.Client.Scan(scanInput)
	if err != nil {
		return []Channel{}, err
	}

	out := make([]Channel, len(result.Items))
	for i, item := range result.Items {
		channel, err := UnmarshalChannel(item)

		if err != nil {
			return out, err
		}

		out[i] = channel
	}

	return out, nil
}

// AddChannel Adds a new channel by name to the channel DB. Fails if a record
// with that Name already exists
func (db *DynamoDBChannelDatabase) AddChannel(name string) (Channel, error) {
	channel := Channel{
		Name:        name,
		IsConnected: false,
	}
	marshalledChannel, err := dynamodbattribute.MarshalMap(channel)

	if err != nil {
		return channel, err
	}

	expression, err := expression.NewBuilder().WithCondition(
		expression.Name("name").AttributeNotExists(),
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
	_, err = db.Client.PutItem(putItemInput)

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

// UpdateChannel Updates an existing Channel record in the DB. Builds an expression
// based on the given builder by adding attribute existence check on the primary key.
// Returns ChannelNotFoundError if the existence check fails
func (db *DynamoDBChannelDatabase) UpdateChannel(name string, builder expression.Builder) (Channel, error) {
	// Anonymous struct with just the channel name
	channelKey := struct {
		Name string `json:"name"`
	}{Name: name}

	marshalledKey, err := dynamodbattribute.MarshalMap(channelKey)

	if err != nil {
		return Channel{}, err
	}

	expression, err := builder.WithCondition(
		expression.Name("name").AttributeExists(),
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
	output, err := db.Client.UpdateItem(updateItemInput)

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

// DisconnectFromChannel Updates the record corresponding to the named channel by
// setting IsConnected to false, then OziachBot departs from the channel. Does not
// delete the DB record
func (bot *OziachBot) DisconnectFromChannel(name string) error {
	// Expression builder to set IsConnected to false
	builder := expression.NewBuilder().WithUpdate(
		expression.Set(expression.Name("isConnected"), expression.Value(false)),
	)

	log.Println("Attempting to disconnect from", name)
	channel, err := bot.ChannelDB.UpdateChannel(name, builder)

	if err == nil {
		bot.TwitchClient.Depart(channel.Name)
		log.Println("Disconnect successful")
	} else {
		log.Println("Disconnect failed")
	}

	return err
}

// ConnectToChannel Updates an existing channel by setting IsConnected to true.
// Then OziachBot joins the channel if it succeeds in doing so
func (bot *OziachBot) ConnectToChannel(name string) error {
	// Expression builder to set IsConnected to true
	builder := expression.NewBuilder().WithUpdate(
		expression.Set(expression.Name("isConnected"), expression.Value(true)),
	)

	log.Println("Attempting to connect to", name)
	channel, err := bot.ChannelDB.UpdateChannel(name, builder)

	if err == nil {
		bot.TwitchClient.Join(channel.Name)
		log.Println("Connection successful")
	} else {
		log.Println("Connection failed")
	}

	return err
}

// ChangeRSN Updates an existing channel by setting rsn
func (bot *OziachBot) ChangeRSN(name, rsn string) error {
	// Expression builder to set rsn
	builder := expression.NewBuilder().WithUpdate(
		expression.Set(expression.Name("rsn"), expression.Value(rsn)),
	)

	log.Printf("Attempting to change rsn of channel %s to %s", name, rsn)
	_, err := bot.ChannelDB.UpdateChannel(name, builder)
	return err
}

// InitBot Initalizes OziachBot by querying for channels and joining them
func (bot *OziachBot) InitBot() error {
	log.Println("Reading channels from DB")
	channels, err := bot.ChannelDB.GetAllChannels()

	if err != nil {
		return err
	}

	// Join all rooms from the DB query
	log.Println("Joining channels")

	for _, channel := range channels {
		bot.TwitchClient.Join(channel.Name)
	}

	return nil
}

// HandleMessage Main callback method to wrap all actions on a PRIVMSG
func (bot *OziachBot) HandleMessage(channel string, user twitch.User, message twitch.Message) {
	// Only handle message if the user is not a bot and not an ignored user
	if _, ok := ignored[user.Username]; !strings.HasSuffix(user.Username, "bot") && !ok {
		log.Printf("Handling message \"%s\" from channel %s\n", message.Text, channel)
		idx := strings.Index(message.Text, " ")
		if idx == -1 {
			idx = len(message.Text)
		}

		switch message.Text[:idx] {
		case "!lvl", "!level":
			numToks := 3
			tokens := strings.SplitN(message.Text, " ", numToks)
			obUser, _ := bot.ChannelDB.GetChannel(channel)
			if len(tokens) < numToks && (len(tokens) < numToks-1 && obUser.RSN != "") {
				break
			}

			skillName := tokens[1]
			player := obUser.RSN

			if player == "" {
				player = tokens[2]
			}

			if len(player) > 12 {
				player = player[:12]
			}

			go bot.HandleSkillLookup(channel, user.DisplayName, skillName, player)
		case "!total", "!overall":
			numToks := 2
			tokens := strings.SplitN(message.Text, " ", numToks)
			obUser, _ := bot.ChannelDB.GetChannel(channel)
			if len(tokens) < numToks && (len(tokens) < numToks-1 && obUser.RSN != "") {
				break
			}

			skillName := "overall"
			player := obUser.RSN

			if player == "" {
				player = tokens[1]
			}

			if len(player) > 12 {
				player = player[:12]
			}

			go bot.HandleSkillLookup(channel, user.DisplayName, skillName, player)
		}
	}
}

// Say Wrapper for Client.Say that prefixes the text with "/me"
func (bot *OziachBot) Say(channel, text string) {
	formattedText := fmt.Sprintf("/me %s", text)
	bot.TwitchClient.Say(channel, formattedText)
}
