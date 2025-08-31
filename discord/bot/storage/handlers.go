package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/theforgeinitiative/integrations/igloohome"
	"github.com/theforgeinitiative/integrations/mail"
	"github.com/theforgeinitiative/integrations/sfdc"
	"github.com/theforgeinitiative/integrations/sheetlog"
)

type StorageHandler struct {
	SFClient        *sfdc.Client
	Campaigns       map[string]string
	IglooHomeClient *igloohome.Client
	SheetLog        *sheetlog.Client
	MailClient      *mail.Client
	Guilds          map[string]struct {
		ID                string
		DoorbellChannelID string
	}
}

func (h *StorageHandler) UnlockStorage(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	contact, err := h.SFClient.GetContactByDiscordID(uid)
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

	status, err := h.SFClient.GetCampaignMembershipStatus(contact.ID, h.Campaigns["storage"])
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
	lockUnit := i.ApplicationCommandData().Options[0].Name
	for _, l := range h.IglooHomeClient.Locks {
		if l["id"] == lock {
			lockUnit = l["unit"]
			break
		}
	}
	fullName := contact.FirstName + " " + contact.LastName
	code, endDate, err := h.IglooHomeClient.GenerateHourly(lock, fullName, 2*time.Hour)
	if err != nil {
		log.Printf("Failed to generate a token: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	log.Printf("%s generated a storage unlock code", fullName)

	err = h.SheetLog.StorageLog(contact, lockUnit)
	if err != nil {
		log.Printf("Error logging storage code retrieval: %s", err)
	}

	if len(code) == 9 {
		for i := 3; i < len(code); i += 4 {
			code = code[:i] + " " + code[i:]
		}
	}

	dmID, err := sendDM(s, uid, fmt.Sprintf(":unlock: You're in!\n\nEnter code `%s` followed by the :unlock: button to open **unit %s**.\n\nThis code is valid until **%s**\n\n%s", code, lockUnit, endDate.Format(time.Kitchen), h.IglooHomeClient.AdditionalInstructions))
	if err != nil {
		log.Printf("Failed to DM lock code: %s", err)
		followUpMsg := ":woozy_face: Oof! We encountered a problem generating an unlock code. Please try again and ask for help if you're stuck."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &followUpMsg,
		})
		return
	}

	followUpMsg := ":unlock: You're in!"
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &followUpMsg,
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Emoji: &discordgo.ComponentEmoji{
							Name: "📩",
						},
						Label: "Check your DMs",
						Style: discordgo.LinkButton,
						URL:   fmt.Sprintf("https://discordapp.com/channels/@me/%s", dmID),
					},
				},
			},
		},
	})

}

func (h *StorageHandler) RequestStorage(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	contact, err := h.SFClient.GetContactByDiscordID(uid)
	if err != nil {
		log.Printf("Failed to lookup member when unlocking storage: %s", err)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}
	status, err := h.SFClient.GetCampaignMembershipStatus(contact.ID, h.Campaigns["storage"])
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
			Content: ":pause_button: You've already requested storage access. Nothing else to do here!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	cm := h.SFClient.CreateCampaignMember(contact.ID, h.Campaigns["storage"], "Requested")
	if cm == nil {
		log.Printf("Failed to add %s to campaign", contact.DisplayName)
		s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: ":woozy_face: Oof! We encountered a problem requesting access. Please try again and ask for help if you're stuck.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	approvalLink := fmt.Sprintf(h.IglooHomeClient.ApprovalLink, cm.StringField("Id"))
	msg := fmt.Sprintf("%s %s has requested access to the storage units.\nEmail: %s\n\nReview in Salesforce: %s", contact.FirstName, contact.LastName, contact.Email, approvalLink)
	err = h.MailClient.SendMail("Storage Unit Access Request", h.IglooHomeClient.ApprovalEmail, msg)
	if err != nil {
		log.Printf("Failed to send approval email for %s: %s", contact.DisplayName, err)
	}
	log.Printf("%s requested storage unit access", contact.DisplayName)
	s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: ":thumbsup: Got it! Someone will review your storage access request soon.",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

// Helper for sending DMs (copied from bot)
func sendDM(s *discordgo.Session, uid, msg string) (string, error) {
	ch, err := s.UserChannelCreate(uid)
	if err != nil {
		return "", fmt.Errorf("failed to create user channel: %w", err)
	}
	_, err = s.ChannelMessageSend(ch.ID, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send user channel message: %w", err)
	}
	return ch.ID, nil
}
