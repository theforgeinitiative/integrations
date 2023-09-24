package mail

import (
	"fmt"

	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/theforgeinitiative/integrations/reconcile"
)

func (c *Client) SendReconcileReport(report reconcile.Report) error {
	text, err := report.RenderText()
	if err != nil {
		return fmt.Errorf("failed to render report: %w", err)
	}
	email := mail.NewSingleEmailPlainText(c.From, "TFI Integrations Reconciliation Report", c.ReportTo, string(text))
	_, err = c.SendgridClient.Send(email)
	return err
}
