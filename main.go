package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gempir/go-twitch-irc"

	"github.com/mfboulos/oziachbot/bot"
	"github.com/mfboulos/oziachbot/hiscores"
)

func main() {
	// Twitch IRC client configuration
	log.Println("Configuring Twitch IRC client")

	twitchClient := twitch.NewClient(
		"OziachBot",
		fmt.Sprintf("oauth:%s", os.Getenv("OZIACH_AUTH")),
	)

	// DynamoDB client connection
	log.Println("Configuring DynamoDB session")

	credentialProvider := &credentials.EnvProvider{}
	credentials := credentials.NewCredentials(credentialProvider)
	dbConfig := aws.NewConfig().WithCredentials(credentials).WithRegion("us-west-1")
	session := session.New(dbConfig)
	dbClient := dynamodb.New(session)

	oziachBot := bot.OziachBot{
		TwitchClient: twitchClient,
		ChannelDB: &bot.DynamoDBChannelDatabase{
			Client: dbClient,
		},
		HiscoreAPI: hiscores.NewOSRSHiscoreAPI(),
	}
	go oziachBot.ServeAPI()

	twitchClient.OnNewMessage(oziachBot.HandleMessage)
	err := oziachBot.InitBot()

	if err != nil {
		log.Fatal(err)
	}

	err = oziachBot.Connect()

	if err != nil {
		log.Fatal(err)
	}
}
