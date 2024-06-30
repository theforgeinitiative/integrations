package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
	"github.com/theforgeinitiative/integrations/config"
	"github.com/theforgeinitiative/integrations/discord/bot"
	"github.com/theforgeinitiative/integrations/groups"
	"github.com/theforgeinitiative/integrations/igloohome"
	"github.com/theforgeinitiative/integrations/mail"
	"github.com/theforgeinitiative/integrations/sfdc"
	"github.com/theforgeinitiative/integrations/sheetlog"
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

	// google groups client
	gc, err := groups.NewClient(viper.GetString("groups.future.email"))
	if err != nil {
		log.Fatalf("Groups client err: %s", err)
	}

	// igloohome client
	ih := igloohome.NewClient(viper.GetString("storage.clientId"), viper.GetString("storage.clientSecret"), viper.GetStringMapString("storage.locks"))
	ih.ApprovalEmail = viper.GetString("storage.approvalEmail")
	ih.ApprovalLink = viper.GetString("storage.approvalLink")
	ih.AdditionalInstructions = viper.GetString("storage.additionalInstructions")

	// Google Sheets log client
	sl, err := sheetlog.NewClient(viper.GetString("storage.log.sheetId"), viper.GetString("storage.log.sheetName"))
	if err != nil {
		log.Fatalf("Sheet client err: %s", err)
	}

	// email client
	mc := mail.NewClient(viper.GetString("mail.apiKey"), viper.GetString("mail.fromName"), viper.GetString("mail.fromEmail"), viper.GetString("mail.fromEmail"))

	// register handlers/commands
	botClient := bot.Bot{
		Session:         sess,
		SFClient:        &sfClient,
		GroupClient:     &gc,
		ID:              viper.GetString("discord.botId"),
		Campaigns:       viper.GetStringMapString("sfdc.campaigns"),
		IglooHomeClient: ih,
		SheetLog:        &sl,
		MailClient:      &mc,
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
