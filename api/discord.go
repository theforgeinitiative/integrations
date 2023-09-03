package api

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"html/template"

	"github.com/bwmarrin/discordgo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/mrz1836/go-sanitize"
	"github.com/theforgeinitiative/integrations/sfdc"
	"go.step.sm/crypto/randutil"
)

//go:embed discord.html
var discordTemplate string

func (h *Handlers) LinkRoleRedirect(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessionOpts

	state, err := randutil.Alphanumeric(32)
	if err != nil {
		return fmt.Errorf("failed to generate state string: %w", err)
	}

	sess.Values["state"] = state
	sess.Save(c.Request(), c.Response())

	return c.Redirect(307, h.DiscordClient.OAuthConfig.AuthCodeURL(state))
}

func (h *Handlers) LinkRoleCallback(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessionOpts

	state, ok := sess.Values["state"]
	if !ok {
		return errors.New("no session state present")
	}
	if state != c.QueryParam("state") {
		return errors.New("session state did not match state in OAuth response")
	}

	token, err := h.DiscordClient.OAuthConfig.Exchange(c.Request().Context(), c.QueryParam("code"))
	if err != nil {
		return fmt.Errorf("failed to retrieve token with auth code: %w", err)
	}

	// Construct a temporary session with user's OAuth2 access_token
	// TODO: Move this into the discord package?
	ts, _ := discordgo.New("Bearer " + token.AccessToken)

	// Retrive user data
	u, err := ts.User("@me")
	if err != nil {
		return fmt.Errorf("failed to retrieve user data: %w", err)
	}

	// Store user id in session too
	sess.Values["discord_id"] = u.ID
	sess.Save(c.Request(), c.Response())

	// write to DB
	err = h.DBClient.SetDiscordUserAuth(u.ID, token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to store refresh token in DB: %w", err)
	}

	return c.Redirect(307, "/discord/")
}

func (h *Handlers) LinkRoleRegister(c echo.Context) error {
	err := h.DiscordClient.RegisterMetadata()
	if err != nil {
		return err
	}
	return c.NoContent(204)
}

func (h *Handlers) VerifyByIDs(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessionOpts

	hid := sanitize.AlphaNumeric(c.FormValue("hid"), false)
	pid := sanitize.AlphaNumeric(c.FormValue("pid"), false)
	if len(hid) == 0 || len(pid) == 0 {
		return echo.NewHTTPError(400, "Both HID and PID must be specified")
	}

	contact, err := h.SFClient.FindContactByIDs(hid, pid)
	if err != nil {
		return fmt.Errorf("failed to get SFDC contact: %w", err)
	}

	// Store minimal contact in session
	sess.Values["contact"] = contact
	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}
	return c.Redirect(303, "/discord/")
}

// NOTE: not currently used
func (h *Handlers) MemberAppCallback(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessionOpts

	contactID := sanitize.AlphaNumeric(c.QueryParam("contact"), false)
	if len(contactID) == 0 {
		return echo.NewHTTPError(400, "contact ID not present in request")
	}

	contact, err := h.SFClient.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("failed to get SFDC contact: %w", err)
	}

	// Store minimal contact in session
	sess.Values["contact"] = contact
	err = sess.Save(c.Request(), c.Response())
	if err != nil {
		return err
	}
	return c.Redirect(303, "/discord/")
}

type DiscordTemplate struct {
	Error    string
	Complete bool
}

func (h *Handlers) DiscordLanding(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options = &sessionOpts

	contact, contactOk := sess.Values["contact"].(sfdc.Contact)
	discordID, discordOk := sess.Values["discord_id"].(string)

	// initialize template
	tmpl, err := template.New("discord").Parse(discordTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	landing := DiscordTemplate{
		Error: c.QueryParam("error"),
	}

	switch {
	case contactOk && discordOk:
		// Update discord user with contact
		h.DBClient.SetDiscordUserContact(discordID, contact.ID)
		metadata := map[string]string{
			//"membership_end": contact.MembershipEndDate,
			"membership_end": "2024-06-30",
		}

		// Get discord auth from DB
		tokens, err := h.DBClient.DiscordTokensByContact(contact.ID)
		if err != nil {
			return fmt.Errorf("failed to get discord tokens from db: %w", err)
		}

		// Update metadata for clients
		fmt.Printf("%+v", tokens)
		err = h.DiscordClient.UpdateMetadata(tokens, metadata)
		if err != nil {
			return fmt.Errorf("failed to write metadata to discord: %w", err)
		}

		landing.Complete = true
	case !discordOk:
		// let's redirect to discord first
		return c.Redirect(307, "/discord/redirect")
	}

	// TODO: Refactor this to use echo's template engine
	var rendered []byte
	buf := bytes.NewBuffer(rendered)
	err = tmpl.ExecuteTemplate(buf, "discord", landing)
	if err != nil {
		return err
	}

	return c.HTMLBlob(200, buf.Bytes())
}
