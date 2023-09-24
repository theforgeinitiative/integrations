package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/sfdc"
)

type Bot struct {
	Session  *discordgo.Session
	SFClient *sfdc.Client
	ID       string
	Guilds   map[string]discord.Guild
}

const unknownMemberErrorCode = 10007

var commands = []discordgo.ApplicationCommand{
	{
		Name:        "link-membership",
		Description: "Link your user to your TFI membership",
	},
	{
		Name:        "welcome",
		Description: "Show welcome message with information about linking membership",
	},
}

var commandsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"link-membership": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.InteractionRespond(i.Interaction, &membershipForm)
		if err != nil {
			panic(err)
		}
	},
	"welcome": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content:    welcomeSlashText,
				Components: welcomeComponents,
			},
		})
		if err != nil {
			panic(err)
		}
	},
}

func (b *Bot) RegisterHandlers() {
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})
	b.Session.AddHandler(b.newMemberHandler)
	b.Session.AddHandler(b.interactionHandler)
}

func (b *Bot) RegisterCommands() {

	//cmdIDs := make(map[string]string, len(commands))

	for _, cmd := range commands {
		_, err := b.Session.ApplicationCommandCreate(b.ID, "", &cmd)
		if err != nil {
			log.Fatalf("Cannot create slash command %q: %v", cmd.Name, err)
		}

		//cmdIDs[rcmd.ID] = rcmd.Name
	}
}
