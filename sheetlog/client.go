package sheetlog

import (
	"context"
	"fmt"
	"time"

	"github.com/theforgeinitiative/integrations/sfdc"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	svc             *sheets.Service
	SheetID         string
	SpreadsheetName string
}

const logDateFormat = "2006-01-02 03:04:05PM"

func NewClient(id, name string) (Client, error) {
	sheetsSvc, err := sheets.NewService(context.TODO(), option.WithScopes("https://www.googleapis.com/auth/spreadsheets"))
	if err != nil {
		return Client{}, fmt.Errorf("failed to create admin service: %w", err)
	}

	return Client{svc: sheetsSvc, SheetID: id, SpreadsheetName: name}, nil
}

func (c *Client) StorageLog(contact sfdc.Contact, lock string) error {
	row := &sheets.ValueRange{
		Values: [][]interface{}{{time.Now().Format(logDateFormat), lock, contact.FirstName, contact.LastName, contact.ID}},
	}

	resp, err := c.svc.Spreadsheets.Values.Append(c.SheetID, c.SpreadsheetName, row).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Do()
	if err != nil {
		return fmt.Errorf("failed to append to sheet: %w", err)
	}
	if resp.HTTPStatusCode != 200 {
		return fmt.Errorf("got non-OK status: %d", resp.HTTPStatusCode)
	}
	return nil
}
