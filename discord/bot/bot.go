package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/groups"
	"github.com/theforgeinitiative/integrations/igloohome"
	"github.com/theforgeinitiative/integrations/mail"
	"github.com/theforgeinitiative/integrations/mq"
	"github.com/theforgeinitiative/integrations/sfdc"
	"github.com/theforgeinitiative/integrations/sheetlog"
)

type Bot struct {
	Session         *discordgo.Session
	SFClient        *sfdc.Client
	GroupClient     *groups.Client
	ID              string
	Guilds          map[string]discord.Guild
	Campaigns       map[string]string
	IglooHomeClient *igloohome.Client
	SheetLog        *sheetlog.Client
	MailClient      *mail.Client
	MQClient        *mq.Client
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
	{
		Name:        "letmein",
		Description: "Rings the doorbell in the LOFT",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "door",
				Type:        discordgo.ApplicationCommandOptionType(discordgo.StringSelectMenu),
				Required:    true,
				Description: "Which door are you at?",
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Front Door",
						Value: "front",
					},
					{
						Name:  "Back Door",
						Value: "back",
					},
				},
			},
		},
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
	for _, cmd := range commands {
		_, err := b.Session.ApplicationCommandCreate(b.ID, "", &cmd)
		if err != nil {
			log.Fatalf("Cannot create slash command %q: %v", cmd.Name, err)
		}
	}

	// storage is special since we're reading these choices from config
	storageCommand := discordgo.ApplicationCommand{
		Name:        "unlock-storage",
		Description: "Generates a temporary code for a TFI storage unit",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "unit",
				Type:        discordgo.ApplicationCommandOptionType(discordgo.StringSelectMenu),
				Required:    true,
				Description: "Which storage unit needs to be unlocked?",
				Choices:     []*discordgo.ApplicationCommandOptionChoice{},
			},
		},
	}
	for _, lock := range b.IglooHomeClient.Locks {
		storageCommand.Options[0].Choices = append(storageCommand.Options[0].Choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  lock["label"],
			Value: lock["id"],
		})
		fmt.Println(lock)
	}
	_, err := b.Session.ApplicationCommandCreate(b.ID, "", &storageCommand)
	if err != nil {
		log.Fatalf("Cannot create slash command %q: %v", "unlock-storage", err)
	}
}
