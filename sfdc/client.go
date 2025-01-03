package sfdc

import (
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/simpleforce/simpleforce"
)

// 100 day grace period
const MembershipGracePeriod = 180 * (24 * time.Hour)

const DateFormat = "2006-01-02"

const authSessionLength = 1 * time.Hour

type Client struct {
	SFClient          *simpleforce.Client
	clientSecret      string
	lastAuthenticated time.Time
}

func NewClient(url, clientID, clientSecret string) (Client, error) {
	sfc := simpleforce.NewClient(url, clientID, simpleforce.DefaultAPIVersion)
	err := sfc.LoginClientCredentials(clientSecret)
	if err != nil {
		return Client{}, fmt.Errorf("error making salesforce client: %w", err)
	}
	c := Client{
		SFClient:          sfc,
		clientSecret:      clientSecret,
		lastAuthenticated: time.Now(),
	}
	return c, nil
}

func (c *Client) Authenticate() error {
	err := c.SFClient.LoginClientCredentials(c.clientSecret)
	if err == nil {
		c.lastAuthenticated = time.Now()
	}
	return err
}

type Contact struct {
	ID                string
	AccountID         string
	Barcode           string
	DisplayName       string
	FirstName         string
	LastName          string
	MembershipEndDate string
	WaiversSignedDate string
	Email             string
	GroupEmail        string
	GroupEmailAlt     string
	DiscordID         string
	MembershipStatus  string
}

func (c Contact) CurrentMember() bool {
	return c.MembershipStatus == "Current" || c.MembershipStatus == "Grace Period"
}

func init() {
	gob.Register(Contact{})
}

func (c *Client) GetContact(id string) (Contact, error) {
	if c.lastAuthenticated.Add(authSessionLength).Before(time.Now()) {
		c.Authenticate()
	}

	// Get an SObject with given type and external ID
	obj := c.SFClient.SObject("Contact").Get(id)
	if obj == nil {
		// Object doesn't exist, handle the error
		return Contact{}, fmt.Errorf("unable to find contact")
	}

	return contactFromSObj(*obj), nil
}

func (c *Client) FindContactByIDs(hid, pid string) (Contact, error) {
	where := fmt.Sprintf(`TFI_Household_ID_ctct__c = '%s'
        AND TFI_Personal_ID__c = '%s'`, hid, pid)
	contacts, err := c.queryContacts(where)
	if err != nil {
		return Contact{}, err
	}
	if len(contacts) < 1 {
		return Contact{}, fmt.Errorf("unable to find contact")
	}
	return contacts[0], nil
}

func (c *Client) FindCurrentMembers() ([]Contact, error) {
	where := `Account.npsp__Membership_Status__c IN ('Current', 'Grace Period')
        AND ( NOT Name LIKE '%%test%%' )`
	return c.queryContacts(where)
}

func (c *Client) GetContactByDiscordID(discordID string) (Contact, error) {
	where := fmt.Sprintf("Discord_ID__c = '%s'", discordID)
	contacts, err := c.queryContacts(where)
	if err != nil {
		return Contact{}, err
	}
	if len(contacts) < 1 {
		return Contact{}, fmt.Errorf("unable to find contact")
	}
	return contacts[0], nil
}

func (c *Client) GetCampaignMembershipStatus(contactID, campaignID string) (string, error) {
	q := fmt.Sprintf(`
	SELECT
		Status
	FROM
		CampaignMember
	WHERE
		CampaignId = '%s' AND ContactId = '%s'
	`, campaignID, contactID)

	result, err := c.SFClient.Query(q)
	if err != nil {
		return "", fmt.Errorf("error running SOQL query: %s", err)
	}
	if len(result.Records) == 0 {
		return "", nil
	}
	return result.Records[0].StringField("Status"), nil
}

func (c *Client) CreateCampaignMember(contactID, campaignID, status string) *simpleforce.SObject {
	if c.lastAuthenticated.Add(authSessionLength).Before(time.Now()) {
		c.Authenticate()
	}
	return c.SFClient.SObject("CampaignMember").
		Set("ContactId", contactID).
		Set("CampaignId", campaignID).
		Set("Status", status).
		Create()
}

func (c *Client) SetDiscordID(contactID, discordID string) error {
	if c.lastAuthenticated.Add(authSessionLength).Before(time.Now()) {
		c.Authenticate()
	}
	updateObj := c.SFClient.SObject("Contact").
		Set("Id", contactID).
		Set("Discord_ID__c", discordID).
		Update()

	if updateObj == nil {
		return errors.New("failed to update contact")
	}

	return nil
}

func (c *Client) queryContacts(where string) ([]Contact, error) {
	if c.lastAuthenticated.Add(authSessionLength).Before(time.Now()) {
		c.Authenticate()
	}

	q := fmt.Sprintf(`
    SELECT
        Id,
		Account.Id,
		TFI_Barcode_for_Button__c,
        TFI_Display_Name_for_Button__c,
		FirstName,
		LastName,
        npo02__MembershipEndDate__c,
        Waivers_signed_date__c,
		Email,
		Google_group__c,
		Google_group_email_2ndary__c,
		Discord_ID__c,
		Account.npsp__Membership_Status__c
    FROM
        Contact 
	WHERE
		%s
	`, where)
	result, err := c.SFClient.Query(q)
	if err != nil {
		return nil, fmt.Errorf("error running SOQL query: %s", err)
	}
	var contacts []Contact
	for _, obj := range result.Records {
		contacts = append(contacts, contactFromSObj(obj))
	}
	return contacts, nil
}

func contactFromSObj(obj simpleforce.SObject) Contact {
	return Contact{
		ID:                obj.StringField("Id"),
		Barcode:           obj.StringField("TFI_Barcode_for_Button__c"),
		DisplayName:       obj.StringField("TFI_Display_Name_for_Button__c"),
		FirstName:         obj.StringField("FirstName"),
		LastName:          obj.StringField("LastName"),
		MembershipEndDate: obj.StringField("npo02__MembershipEndDate__c"),
		WaiversSignedDate: obj.StringField("Waivers_signed_date__c"),
		Email:             obj.StringField("Email"),
		GroupEmail:        obj.StringField("Google_group__c"),
		GroupEmailAlt:     obj.StringField("Google_group_email_2ndary__c"),
		DiscordID:         obj.StringField("Discord_ID__c"),
		MembershipStatus:  obj.SObjectField("Account", "Account").StringField("npsp__Membership_Status__c"),
	}
}
