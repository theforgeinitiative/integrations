package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "join-future-forge" {
			err := s.InteractionRespond(i.Interaction, b.joinFutureCommandHandler(s, i))
			if err != nil {
				panic(err)
			}
			return
		}
		if i.ApplicationCommandData().Name == "unlock-storage" {
			b.unlockStorageHandler(s, i)
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
		}
	case discordgo.InteractionModalSubmit:
		var err error
		switch i.ModalSubmitData().CustomID {
		case "link_membership_form":
			err = b.linkMembershipHadler(s, i)
		case "join_future_form":
			err = b.addFutureForgeHandler(s, i)
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

func (b *Bot) joinFutureCommandHandler(_ *discordgo.Session, i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	contact, err := b.SFClient.GetContactByDiscordID(i.Member.User.ID)
	var email string
	if err == nil {
		email = contact.GroupEmail
	}
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "join_future_form",
			Title:    "Join the Future Forge group",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "email",
							Label:       "Email Address (GMail/Google Account)",
							Style:       discordgo.TextInputShort,
							Placeholder: "somebody@gmail.com",
							Required:    true,
							Value:       email,
						},
					},
				},
			},
		},
	}
}

func (b *Bot) unlockStorageHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Generating an unlock code... :thinking:",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	var uid string
	if i.Member != nil {
		uid = i.Member.User.ID
	} else {
		uid = i.User.ID
	}
	contact, err := b.SFClient.GetContactByDiscordID(uid)
	if err != nil {
		log.Printf("Failed to lookup member when unlocking storage: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	status, err := b.SFClient.GetCampaignMembershipStatus(contact.ID, b.Campaigns["storage"])
	if err != nil {
		log.Printf("Failed to retrieve campaign membership status: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	if status != "Approved" {
		log.Printf("%s tried to unlock storage, but was not an approved campaign member", contact.DisplayName)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":police_officer: You need to be approved to unlock storage. Please reach out to TFI Ops if you believe you should have access.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	fullName := contact.FirstName + " " + contact.LastName

	code, endDate, err := b.IglooHomeClient.GenerateHourly(fullName, 2*time.Hour)
	if err != nil {
		log.Printf("Failed to generate an OTP: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	log.Printf("%s generated a storage unlock code", fullName)

	// TODO: Google Sheet stuff
	//s.ChannelMessageSend("1255309795997520063", fmt.Sprintf("**%s** requested a storage unit PIN.", contact.DisplayName))
	err = b.SheetLog.StorageLog(fullName)
	if err != nil {
		log.Printf("Error logging storage code retrieval: %s", err)
	}
	if len(code) == 9 {
		for i := 3; i < len(code); i += 4 {
			code = code[:i] + " " + code[i:]
		}
	}
	s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: fmt.Sprintf(":unlock: You're in!\n\nEnter code `%s` to unlock the storage unit.\n\nThis code is valid until %s", code, endDate.Format(time.Kitchen)),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
