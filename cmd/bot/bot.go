package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
	"github.com/theforgeinitiative/integrations/config"
	"github.com/theforgeinitiative/integrations/discord/bot"
	"github.com/theforgeinitiative/integrations/sfdc"
)

func main() {
	// get config
	err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	// setup Discord session
	sess, err := discordgo.New("Bot " + viper.GetString("discord.botToken"))
	if err != nil {
		log.Fatalf("Failed to create Discord client: %s", err)
	}
	sess.Identify.Intents = discordgo.IntentsGuildMembers

	// setup SFDC connection
	sfClient, err := sfdc.NewClient(viper.GetString("sfdc.url"), viper.GetString("sfdc.clientId"), viper.GetString("sfdc.clientSecret"))
	if err != nil {
		log.Fatalf("Failed to create SFDC client: %s", err)
	}

	// register handlers/commands
	botClient := bot.Bot{
		Session:  sess,
		SFClient: &sfClient,
		ID:       viper.GetString("discord.botId"),
	}

	err = viper.UnmarshalKey("discord.guilds", &botClient.Guilds)
	if err != nil {
		log.Fatal("Failed to read Discord guild config", err)
	}

	botClient.RegisterCommands()
	botClient.RegisterHandlers()

	err = sess.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer sess.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Graceful shutdown")
}
