package storage

import "github.com/bwmarrin/discordgo"

func (h *StorageHandler) RegisterCommand(session *discordgo.Session, botID string) error {
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
	for _, lock := range h.IglooHomeClient.Locks {
		storageCommand.Options[0].Choices = append(storageCommand.Options[0].Choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  lock["label"],
			Value: lock["id"],
		})
	}
	_, err := session.ApplicationCommandCreate(botID, "", &storageCommand)
	if err != nil {
		return err
	}
	return nil
}
