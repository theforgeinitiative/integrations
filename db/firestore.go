package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const DiscordUserCollection = "discord_users"

type Client struct {
	FirestoreClient *firestore.Client
}

func NewClient(projectID string) (*Client, error) {
	fs, err := firestore.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Firestore client: %w", err)
	}

	return &Client{
		FirestoreClient: fs,
	}, nil
}

func (c *Client) SetDiscordUserContact(userID string, ContactID string) error {
	_, err := c.FirestoreClient.Collection(DiscordUserCollection).Doc(userID).Set(context.Background(), map[string]interface{}{
		"contact_id": ContactID,
	}, firestore.MergeAll)

	return err
}

func (c *Client) SetDiscordUserAuth(userID string, refreshToken string) error {
	_, err := c.FirestoreClient.Collection(DiscordUserCollection).Doc(userID).Set(context.Background(), map[string]interface{}{
		"refresh_token": refreshToken,
	}, firestore.MergeAll)

	return err
}

func (c *Client) DiscordTokensByContact(contact string) (map[string]string, error) {
	iter := c.FirestoreClient.Collection(DiscordUserCollection).Where("contact_id", "==", contact).Documents(context.Background())
	tokens := make(map[string]string)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		tokens[doc.Ref.ID] = data["refresh_token"].(string)
	}

	return tokens, nil
}
