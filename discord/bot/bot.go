package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/groups"
	"github.com/theforgeinitiative/integrations/igloohome"
	"github.com/theforgeinitiative/integrations/mail"
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
}

const unknownMemberErrorCode = 10007

var commands = []discordgo.ApplicationCommand{
	{
		Name:        "link-membership",
		Description: "Link your user to your TFI membership",
	},
	{
		Name:        "join-future-forge",
		Description: "Join the Future Forge Google Group",
	},
	{
		Name:        "unlock-storage",
		Description: "Generates a temporary code for a TFI storage unit",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "unit",
				Type:        discordgo.ApplicationCommandOptionType(discordgo.StringSelectMenu),
				Required:    true,
				Description: "Which storage unit needs to be unlocked?",
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "1501 - Outside",
						Value: "1501",
					},
					{
						Name:  "2091 - CNC and things",
						Value: "2091",
					},
					{
						Name:  "2099 - FTC, FLL, CCR",
						Value: "2099",
					},
					{
						Name:  "2100 - To Be Determined",
						Value: "2100",
					},
					{
						Name:  "2116 - Team Building, Tools",
						Value: "2116",
					},
					{
						Name:  "2166 - Outreach, PyroTech",
						Value: "2166",
					},
				},
			},
		},
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
