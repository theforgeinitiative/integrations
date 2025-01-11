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
		if i.ApplicationCommandData().Name == "unlock-storage" {
			b.unlockStorageHandler(s, i)
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
			b.requestStorageHandler(s, i)
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
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please ensure you've linked your membership to your Discord account and try again.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	if !contact.CurrentMember() {
		log.Printf("%s tried to unlock storage, but was not a current member", contact.DisplayName)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":customs: You must be a current member to access TFI storage.",
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

	if status == "" {
		log.Printf("%s tried to unlock storage, but was not yet in the campaign", contact.DisplayName)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":octagonal_sign: You need to be approved to unlock storage. If you believe you require access, request it with the button below.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Emoji: &discordgo.ComponentEmoji{
								Name: "🔑",
							},
							Label:    "Request Access",
							Style:    discordgo.PrimaryButton,
							CustomID: "storage_request_access",
						},
					},
				},
			},
		})
		return
	}

	if status != "Approved" {
		log.Printf("%s tried to unlock storage, but was not an approved campaign member", contact.DisplayName)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":customs: Your request for storage access is still awaiting approval.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	lock := i.ApplicationCommandData().Options[0].StringValue()
	fullName := contact.FirstName + " " + contact.LastName
	code, endDate, err := b.IglooHomeClient.GenerateHourly(lock, fullName, 2*time.Hour)
	if err != nil {
		log.Printf("Failed to generate a token: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	log.Printf("%s generated a storage unlock code", fullName)

	// TODO: Google Sheet stuff
	err = b.SheetLog.StorageLog(contact, lock)
	if err != nil {
		log.Printf("Error logging storage code retrieval: %s", err)
	}

	// pretty format the code
	if len(code) == 9 {
		for i := 3; i < len(code); i += 4 {
			code = code[:i] + " " + code[i:]
		}
	}

	// Send the user a DM with code
	err = b.sendDM(uid, fmt.Sprintf(":unlock: You're in!\n\nEnter code `%s` followed by the :unlock: button to open **unit %s**.\n\nThis code is valid until **%s**\n\n%s", code, lock, endDate.Format(time.Kitchen), b.IglooHomeClient.AdditionalInstructions))
	if err != nil {
		log.Printf("Failed to DM lock code: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
	}
}

func (b *Bot) requestStorageHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Requesting access... :thinking:",
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
			Content: ":woozy_face: Oof! We encountered a problem. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	status, err := b.SFClient.GetCampaignMembershipStatus(contact.ID, b.Campaigns["storage"])
	if err != nil {
		log.Printf("Failed to retrieve campaign membership status: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	if status != "" {
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":pause_button: You've alredy requested storage access. Nothing else to do here!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	cm := b.SFClient.CreateCampaignMember(contact.ID, b.Campaigns["storage"], "Requested")
	if cm == nil {
		log.Printf("Failed to add %s to campaign", contact.DisplayName)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem requesting access. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	approvalLink := fmt.Sprintf(b.IglooHomeClient.ApprovalLink, cm.StringField("Id"))
	msg := fmt.Sprintf("%s %s has requested access to the storage units.\nEmail: %s\n\nReview in Salesforce: %s", contact.FirstName, contact.LastName, contact.Email, approvalLink)
	err = b.MailClient.SendMail("Storage Unit Access Request", b.IglooHomeClient.ApprovalEmail, msg)
	if err != nil {
		log.Printf("Failed to send approval email for %s: %s", contact.DisplayName, err)
	}
	log.Printf("%s requested storage unit access", contact.DisplayName)
	s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: ":thumbsup: Got it! Someone will review your storage access request soon.",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
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
