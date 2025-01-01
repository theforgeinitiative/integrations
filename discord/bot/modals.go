package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) linkMembershipHadler(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Verifying your membership... :thinking:",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return err
	}

	// lookup by HID and PID
	data := i.ModalSubmitData()
	hid := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	pid := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	contact, err := b.SFClient.FindContactByIDs(hid, pid)
	if err != nil {
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "Sorry, I wasn't able to find your HID/PID. Use the recovery link below if you can't find it.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Emoji: discordgo.ComponentEmoji{
								Name: "📱",
							},
							Label: "Member App",
							Style: discordgo.LinkButton,
							URL:   "https://app.theforgeinitiative.org",
						},
						discordgo.Button{
							Emoji: discordgo.ComponentEmoji{
								Name: "🔍",
							},
							Label: "ID Recovery Form",
							Style: discordgo.LinkButton,
							URL:   "https://www.tfaforms.com/4764292",
						},
					},
				},
			},
		})
		return fmt.Errorf("failed to find contact for hid/pid %s/%s: %w", hid, pid, err)
	}

	err = b.SFClient.SetDiscordID(contact.ID, i.Member.User.ID)
	if err != nil {
		log.Printf("failed to update Discord ID for contact %s: %s", contact.ID, err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: "I had trouble linking your membership, but it's probably not your fault. Please try again.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return fmt.Errorf("failed to send follow-up response: %s", err)
	}

	// Set role
	for gName, guild := range b.Guilds {
		err := s.GuildMemberRoleAdd(guild.ID, i.Member.User.ID, guild.MemberRoleID)
		restErr, ok := err.(*discordgo.RESTError)
		// skip this guild if member isn't part of it
		if ok && restErr.Message.Code == unknownMemberErrorCode {
			continue
		}
		if err != nil {
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "I encountered an error trying to give you a role. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return fmt.Errorf("failed to add role for %s in guild %s: %s", memberDisplayName(i.Member), gName, err)
		}
		log.Printf("Successfully added member role for %s in guild %s", memberDisplayName(i.Member), gName)
	}

	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: ":tada: You're all set! If you didn't already have access, check out all the new member areas.",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		return fmt.Errorf("failed to send success response: %s", err)
	}
	return nil
}
