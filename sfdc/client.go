package sfdc

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/simpleforce/simpleforce"
)

// Eventually make this configurable/dynamic
const MembershipYearStart = "2023-07-01"

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
	DisplayName       string
	MembershipEndDate string
	WaiversSignedDate string
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

	contact := Contact{
		ID:                obj.StringField("Id"),
		DisplayName:       obj.StringField("TFI_Display_Name_for_Button__c"),
		MembershipEndDate: obj.StringField("npo02__MembershipEndDate__c"),
		WaiversSignedDate: obj.StringField("Waivers_signed_date__c"),
	}
	return contact, nil
}

func (c *Client) FindContactByIDs(hid, pid string) (Contact, error) {
	if c.lastAuthenticated.Add(authSessionLength).Before(time.Now()) {
		c.Authenticate()
	}

	q := fmt.Sprintf(`
    SELECT
        Id,
        TFI_Display_Name_for_Button__c,
        npo02__MembershipEndDate__c,
        Waivers_signed_date__c 
    FROM
        Contact 
    WHERE
	    TFI_Household_ID_ctct__c = '%s'
        AND TFI_Personal_ID__c = '%s'
	`, hid, pid)
	result, err := c.SFClient.Query(q)
	if err != nil {
		return Contact{}, fmt.Errorf("error running SOQL query: %s", err)
	}
	if len(result.Records) < 1 {
		return Contact{}, fmt.Errorf("unable to find contact")
	}
	obj := result.Records[0]
	contact := Contact{
		ID:                obj.StringField("Id"),
		DisplayName:       obj.StringField("TFI_Display_Name_for_Button__c"),
		MembershipEndDate: obj.StringField("npo02__MembershipEndDate__c"),
		WaiversSignedDate: obj.StringField("Waivers_signed_date__c"),
	}
	return contact, nil
}
