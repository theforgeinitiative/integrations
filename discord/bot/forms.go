package bot

import "github.com/bwmarrin/discordgo"

const welcomeHelp = `**:link: Link your membership:**
To keep this a safe space for Forge members of all ages, many of our channels are only open to members. To access them, click the button below to link your Discord user to your Forge membership. You will need your Household and Personal IDs which can be found in the Member App or recovered via the form below.`

const welcomeMsg = `Welcome to the TFI Discord server, <@%s>!

` + welcomeHelp

const welcomeSlashText = `Welcome to the TFI Discord server!

` + welcomeHelp

var welcomeComponents = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Emoji: discordgo.ComponentEmoji{
					Name: "🔗",
				},
				Label:    "Link Membership",
				Style:    discordgo.PrimaryButton,
				CustomID: "show_link_form",
			},
		},
	},
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
}

var membershipForm = discordgo.InteractionResponse{
	Type: discordgo.InteractionResponseModal,
	Data: &discordgo.InteractionResponseData{
		CustomID: "link_membership_form",
		Title:    "Link to your TFI membership",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "hid",
						Label:       "Household ID",
						Style:       discordgo.TextInputShort,
						Placeholder: "H01234",
						Required:    true,
						MaxLength:   6,
						MinLength:   6,
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "pid",
						Label:       "Personal ID",
						Style:       discordgo.TextInputShort,
						Placeholder: "P012345",
						Required:    true,
						MaxLength:   7,
						MinLength:   7,
					},
				},
			},
		},
	},
}

var Storage = []discordgo.MessageComponent{
	discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Emoji: discordgo.ComponentEmoji{
					Name: "🔗",
				},
				Label:    "Link Membership",
				Style:    discordgo.PrimaryButton,
				CustomID: "show_link_form",
			},
		},
	},
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
}
