package mail

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type Client struct {
	SendgridClient *sendgrid.Client
	From           *mail.Email
	ReportTo       *mail.Email
}

func NewClient(apiKey, senderName, senderEmail, reportEmail string) Client {
	return Client{
		SendgridClient: sendgrid.NewSendClient(apiKey),
		From:           mail.NewEmail(senderName, senderEmail),
		ReportTo:       mail.NewEmail("", reportEmail),
	}
}
