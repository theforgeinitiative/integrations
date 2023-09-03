package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"
)

type TokenStore interface {
	SetDiscordUserAuth(userID string, refreshToken string) error
}

type Client struct {
	BotSession  *discordgo.Session
	OAuthConfig *oauth2.Config
	Store       TokenStore
}

func (c *Client) RegisterMetadata() error {
	_, err := c.BotSession.ApplicationRoleConnectionMetadataUpdate(c.OAuthConfig.ClientID, []*discordgo.ApplicationRoleConnectionMetadata{
		{
			Type:        discordgo.ApplicationRoleConnectionMetadataDatetimeGreaterThanOrEqual,
			Key:         "membership_end",
			Name:        "Member Until",
			Description: "Membership end date",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to register Discord role connection metadata: %w", err)
	}
	return nil
}

func (c *Client) UpdateMetadata(refreshTokens map[string]string, metadata map[string]string) error {
	for id, refresh := range refreshTokens {
		accessToken, err := c.getAccessToken(id, refresh)
		if err != nil {
			return err
		}
		ts, _ := discordgo.New("Bearer " + accessToken)
		_, err = ts.UserApplicationRoleConnectionUpdate(c.OAuthConfig.ClientID, &discordgo.ApplicationRoleConnection{
			PlatformName:     "The Forge Initiative",
			PlatformUsername: "TFI Member Bot",
			Metadata:         metadata,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) getAccessToken(id string, refreshToken string) (string, error) {
	restoredToken := &oauth2.Token{
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}
	ts := c.OAuthConfig.TokenSource(context.Background(), restoredToken)
	token, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	// save new refresh token
	err = c.Store.SetDiscordUserAuth(id, token.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token in DB: %w", err)
	}

	return token.AccessToken, nil
}
