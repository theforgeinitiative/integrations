package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "unlock-storage" {
			b.StorageHandler.UnlockStorage(s, i)
			return
		}
		if i.ApplicationCommandData().Name == "letmein" {
			b.letmeinHandler(s, i)
			return
		}
		if h, ok := commandsHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	case discordgo.InteractionMessageComponent:
		switch i.MessageComponentData().CustomID {
		case "show_link_form":
			err := s.InteractionRespond(i.Interaction, &membershipForm)
			if err != nil {
				panic(err)
			}
		case "storage_request_access":
			b.StorageHandler.RequestStorage(s, i)
		}
	case discordgo.InteractionModalSubmit:
		var err error
		switch i.ModalSubmitData().CustomID {
		case "link_membership_form":
			err = b.linkMembershipHadler(s, i)
		}
		if err != nil {
			log.Printf("Failed to handle modal response for %s: %s", i.ModalSubmitData().CustomID, err)
			return
		}
	}
}

func (b *Bot) newMemberHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	log.Printf("got member add event!")

	welcomeChan := b.guildWelcomeChannel(m.GuildID)
	if len(welcomeChan) == 0 {
		log.Printf("Received member add event for unrecognized guild id: %s", m.GuildID)
		return
	}
	// Check if member is already in Salesforce
	contact, err := b.SFClient.GetContactByDiscordID(m.User.ID)
	// handle the case where the contact already exists
	if err == nil {
		for gName, guild := range b.Guilds {
			if len(guild.MemberRoleID) > 0 {
				err := s.GuildMemberRoleAdd(guild.ID, m.User.ID, guild.MemberRoleID)
				restErr, ok := err.(*discordgo.RESTError)
				// skip this guild if member isn't part of it
				if ok && restErr.Message.Code == unknownMemberErrorCode {
					continue
				}
				if err != nil {
					log.Printf("failed to add role for %s in guild %s: %s", memberAddDisplayName(m), gName, err)
					return
				}
				log.Printf("Successfully added member role for %s in guild %s", memberAddDisplayName(m), gName)
				err = s.GuildMemberNickname(guild.ID, m.User.ID, contact.DisplayName)
				if err != nil {
					log.Printf("failed to add role for %s in guild %s: %s", memberAddDisplayName(m), gName, err)
				}
				log.Printf("Successfully set nick for %s in guild %s", memberAddDisplayName(m), gName)
			}
		}
		_, err = s.ChannelMessageSend(welcomeChan, fmt.Sprintf("Welcome, <@%s>! You've already linked your membership, so you're good to go. :sunglasses:", m.User.ID))
		if err != nil {
			log.Printf("Failed to send linked user welcome message: %s", err)
		}
		return
	}

	_, err = s.ChannelMessageSendComplex(welcomeChan, &discordgo.MessageSend{
		Content:    fmt.Sprintf(welcomeMsg, m.User.ID),
		Components: welcomeComponents,
	})
	if err != nil {
		log.Printf("Failed to send new user welcome message: %s", err)
	}
	log.Println("Sent welcome message to new user!")
}

func (b *Bot) guildWelcomeChannel(g string) string {
	for _, guild := range b.Guilds {
		if guild.ID == g {
			return guild.WelcomeChannelID
		}
	}
	return ""
}

func (b *Bot) guildDoorbellChannel(g string) string {
	for _, guild := range b.Guilds {
		if guild.ID == g {
			return guild.DoorbellChannelID
		}
	}
	return b.Guilds["tfi"].DoorbellChannelID
}

func memberDisplayName(m *discordgo.Member) string {
	if len(m.Nick) > 0 {
		return m.Nick
	}
	if len(m.User.GlobalName) > 0 {
		return m.User.GlobalName
	}
	return m.User.Username
}

func memberAddDisplayName(m *discordgo.GuildMemberAdd) string {
	if len(m.Nick) > 0 {
		return m.Nick
	}
	if len(m.User.GlobalName) > 0 {
		return m.User.GlobalName
	}
	return m.User.Username
}

func (b *Bot) letmeinHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Working on it... :thinking:",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	var uid string
	if i.Member != nil {
		uid = i.Member.User.ID
	} else {
		uid = i.User.ID
	}

	door := i.ApplicationCommandData().Options[0].StringValue()

	err := b.MQClient.RingDoorbell(door)
	if err != nil {
		log.Printf("Failed to publish doorbell message: %s", err)
		msg := ":woozy_face: Oof! I encountered a problem requesting access. Please try again, but worst case you may have to ask someone to let you in the old fashioned way."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		return
	}
	followup := "I rang the bell for you! Sit tight... :person_running_facing_right:"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &followup,
	})

	_, err = s.ChannelMessageSendComplex(b.guildDoorbellChannel(i.GuildID), &discordgo.MessageSend{
		Content: fmt.Sprintf(`:bell: Hey everyone! <@%s> needs to be let in the %s door.`, uid, door),
	})
	if err != nil {
		log.Printf("Failed to send doorbell message for %s", memberDisplayName(i.Member))
	}

	log.Printf("Rang the %s doorbell for %s", door, memberDisplayName(i.Member))
}

func (b *Bot) sendDM(uid, msg string) error {
	ch, err := b.Session.UserChannelCreate(uid)
	if err != nil {
		return fmt.Errorf("failed to create user channel: %w", err)
	}
	_, err = b.Session.ChannelMessageSend(ch.ID, msg)
	if err != nil {
		return fmt.Errorf("failed to send user channel message: %w", err)
	}
	return nil
}
