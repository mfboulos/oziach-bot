package bot

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
	"github.com/gempir/go-twitch-irc"
)

// OziachBot Wrapper class for twitch.Client that encompasses all OziachBot features
type OziachBot struct {
	TwitchClient *twitch.Client
	dbClient     *dynamodb.DynamoDB
	dbStream     *dynamodbstreams.DynamoDBStreams
	streamArn    string
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
	dClient := dynamodb.New(session)
	dStream := dynamodbstreams.New(session)

	tableName := "ob-channels"
	scanInput := &dynamodb.ScanInput{
		TableName: &tableName,
	}

	result, err := dClient.Scan(scanInput)
	if err != nil {
		panic(err)
	}

	bot := OziachBot{
		TwitchClient: tClient,
		dbClient:     dClient,
		dbStream:     dStream,
		streamArn:    os.Getenv("OZIACH_CHANNEL_DB"),
	}

	tClient.OnNewMessage(bot.HandleMessage)

	for _, item := range result.Items {
		channel := Channel{}
		dynamodbattribute.UnmarshalMap(item, &channel)
		tClient.Join(channel.Name)
	}

	go bot.listenForChanges()

	return bot
}

// GetLatestShardIterator Gets the shard iterator pointing to the latest record in the stream
func (bot *OziachBot) GetLatestShardIterator() (shardID, shardIterator *string, err error) {
	dsi := (&dynamodbstreams.DescribeStreamInput{})
	dsi.SetStreamArn(bot.streamArn)
	desc, err := bot.dbStream.DescribeStream(dsi)

	if err != nil {
		return nil, nil, err
	}

	shard := desc.StreamDescription.Shards[0]

	iterInput := (&dynamodbstreams.GetShardIteratorInput{
		ShardId: shard.ShardId,
	})
	iterInput = iterInput.SetShardIteratorType(dynamodbstreams.ShardIteratorTypeLatest)
	iterInput = iterInput.SetStreamArn(bot.streamArn)
	shardIteratorOutput, err := bot.dbStream.GetShardIterator(iterInput)
	return shard.ShardId, shardIteratorOutput.ShardIterator, err
}

func (bot *OziachBot) mapFuncToStreamOutput(shardID, shardIterator *string, f func(record *dynamodbstreams.Record)) error {
	if shardIterator != nil {
		gri := &dynamodbstreams.GetRecordsInput{
			ShardIterator: shardIterator,
		}
		recordOutput, err := bot.dbStream.GetRecords(gri)

		if err != nil {
			return err
		}

		for _, record := range recordOutput.Records {
			f(record)
		}

		*shardIterator = *recordOutput.NextShardIterator
	}

	if shardIterator == nil {
		dsi := (&dynamodbstreams.DescribeStreamInput{})
		dsi.SetStreamArn(bot.streamArn)
		dsi.SetExclusiveStartShardId(*shardID)
		desc, err := bot.dbStream.DescribeStream(dsi)

		if err != nil {
			return err
		}

		shards := desc.StreamDescription.Shards[1:]

		for _, shard := range shards {
			*shardID = *shard.ShardId
			iterInput := (&dynamodbstreams.GetShardIteratorInput{})
			iterInput = iterInput.SetShardId(*shardID)
			iterInput = iterInput.SetShardIteratorType(dynamodbstreams.ShardIteratorTypeTrimHorizon)
			iterInput = iterInput.SetStreamArn(bot.streamArn)
			shardIteratorOutput, err := bot.dbStream.GetShardIterator(iterInput)

			if err != nil {
				return err
			}

			shardIterator = shardIteratorOutput.ShardIterator
			gri := &dynamodbstreams.GetRecordsInput{
				ShardIterator: shardIterator,
			}
			recordOutput, err := bot.dbStream.GetRecords(gri)

			if err != nil {
				return err
			}

			for _, record := range recordOutput.Records {
				f(record)
			}
			*shardIterator = *recordOutput.NextShardIterator
		}
	}

	return nil
}

func (bot *OziachBot) listenForChanges() {
	shardID, shardIterator, err := bot.GetLatestShardIterator()
	tick := time.Tick(10 * time.Second)

	if err != nil {
		for range tick {
			shardID, shardIterator, err = bot.GetLatestShardIterator()

			if err == nil {
				break
			}
		}
	}

	for range tick {
		bot.mapFuncToStreamOutput(shardID, shardIterator, func(record *dynamodbstreams.Record) {
			channel := Channel{}
			switch *record.EventName {
			case dynamodbstreams.OperationTypeInsert:
				dynamodbattribute.UnmarshalMap(record.Dynamodb.NewImage, &channel)
				bot.TwitchClient.Join(channel.Name)
			case dynamodbstreams.OperationTypeRemove:
				dynamodbattribute.UnmarshalMap(record.Dynamodb.OldImage, &channel)
				bot.TwitchClient.Depart(channel.Name)
			}
		})
	}
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
