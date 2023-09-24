package groups

import (
	"context"
	"fmt"

	admin "google.golang.org/api/admin/directory/v1"
)

type Client struct {
	adminSvc *admin.Service
	Group    string
}

func NewClient(group string) (Client, error) {
	adminSvc, err := admin.NewService(context.TODO())
	if err != nil {
		return Client{}, fmt.Errorf("failed to create admin service: %w", err)
	}

	return Client{adminSvc: adminSvc, Group: group}, nil
}

func (c *Client) LookupMember(email string) (*admin.Member, error) {
	return c.adminSvc.Members.Get(c.Group, email).Do()
}

func (c *Client) ListMembers() ([]*admin.Member, error) {
	var memberList []*admin.Member
	pageToken := ""
	for {
		listResp, err := c.adminSvc.Members.List(c.Group).MaxResults(200).PageToken(pageToken).Do()
		if err != nil {
			return nil, err
		}
		memberList = append(memberList, listResp.Members...)
		pageToken = listResp.NextPageToken
		if len(pageToken) == 0 {
			break
		}
	}
	return memberList, nil
}

func (c *Client) AddMember(email string) error {
	member := admin.Member{
		Email: email,
	}
	_, err := c.adminSvc.Members.Insert(c.Group, &member).Do()
	return err
}

func (c *Client) RemoveMember(email string) error {
	return c.adminSvc.Members.Delete(c.Group, email).Do()
}
