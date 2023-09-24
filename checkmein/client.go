package checkmein

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/theforgeinitiative/integrations/sfdc"
)

const BulkAddDateFormat = "1/2/2006"

type Client struct {
	URL        string
	Username   string
	Password   string
	httpClient *http.Client
}

type BulkAddMember struct {
	Barcode           string `csv:"TFI Barcode for Button"`
	DisplayName       string `csv:"TFI Display Name for Button"`
	FirstName         string `csv:"First Name"`
	LastName          string `csv:"Last Name"`
	MembershipEndDate string `csv:"Membership End Date"`
	Email             string `csv:"Email"`
}

func NewClient(url, username, password string) Client {
	jar, _ := cookiejar.New(nil)

	return Client{
		URL:      url,
		Username: username,
		Password: password,
		httpClient: &http.Client{
			Jar: jar,
		},
	}
}

func (c *Client) BulkAdd(contacts []sfdc.Contact) error {
	err := c.authenticate()
	if err != nil {
		return fmt.Errorf("failed to authenticate to checkmein: %w", err)
	}

	// Build CSV file
	var rows []BulkAddMember
	for _, c := range contacts {
		endDate, err := time.Parse(sfdc.DateFormat, c.MembershipEndDate)
		if err != nil {
			return fmt.Errorf("failed to parse membership end date for %s: %w", c.DisplayName, err)
		}
		rows = append(rows, BulkAddMember{
			Barcode:           c.Barcode,
			DisplayName:       c.DisplayName,
			FirstName:         c.FirstName,
			LastName:          c.LastName,
			MembershipEndDate: endDate.Format(BulkAddDateFormat),
			Email:             c.Email,
		})
	}

	csvContent, err := gocsv.MarshalBytes(rows)
	if err != nil {
		return fmt.Errorf("failed to generate CSV: %w", err)
	}

	// generate multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("csvfile", "report.csv")
	if err != nil {
		return fmt.Errorf("failed to write csv to upload form: %w", err)
	}
	_, err = fw.Write(csvContent)
	if err != nil {
		return fmt.Errorf("failed to buffer CSV: %w", err)
	}
	w.Close()

	req, err := http.NewRequest("POST", c.URL+"/admin/bulkAddMembers", &b)
	if err != nil {
		return fmt.Errorf("failed to build request for bulk add: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to bulk add to checkmein: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received bad status from checkmein: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) authenticate() error {
	// check if session cookie is still valid
	url, _ := url.Parse(c.URL)
	for _, cookie := range c.httpClient.Jar.Cookies(url) {
		if cookie.Name == "session_id" && cookie.Valid() == nil {
			return nil
		}
	}

	req, err := http.NewRequest("GET", c.URL+"/profile/loginAttempt", nil)
	if err != nil {
		return fmt.Errorf("failed to build login request: %w", err)
	}
	q := req.URL.Query()
	q.Add("username", c.Username)
	q.Add("password", c.Password)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login to checkmein: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received bad status from checkmein: %d", resp.StatusCode)
	}

	return nil
}
