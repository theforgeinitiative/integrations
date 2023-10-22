package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/theforgeinitiative/integrations/reconcile"
	"github.com/theforgeinitiative/integrations/sfdc"
	admin "google.golang.org/api/admin/directory/v1"
)

func (h *Handlers) Reconcile(c echo.Context) error {
	dryRun := true
	if param := c.QueryParam("dry_run"); len(param) > 0 {
		var err error
		dryRun, err = strconv.ParseBool(param)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid format for dry_run param: %s")
		}
	}

	var user string
	if u := c.Get("authorized_user"); u != nil {
		user = u.(string)
	}

	report := reconcile.Report{
		Date:      time.Now(),
		CheckMeIn: true,
		User:      user,
		Discord:   make(map[string]reconcile.Changes),
		Groups:    make(map[string]reconcile.Changes),
	}
	respStatus := http.StatusOK

	// get all current members from SFDC
	contactList, err := h.SFClient.FindCurrentMembers()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve current members from sfdc", err)
	}
	contacts := contactEmailMap(contactList)
	// add exceptions from config
	h.addExceptions(contacts)

	// CHECKMEIN
	if !dryRun {
		err = h.CheckMeInClient.BulkAdd(contactList)
		if err != nil {
			c.Logger().Errorf("Failed to bulk add users to checkmein: %s", err)
			respStatus = http.StatusMultiStatus
			report.CheckMeIn = false
		}
	}

	// GOOGLE GROUPS

	// get all members of Google Group
	emailList, err := h.GroupsClient.ListMembers()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve group members").WithInternal(err)
	}
	emails := groupEmailMap(emailList)

	var errored, add, del []string

	// iterate over Salesforce data to find additions
	for key, contact := range contacts {
		if _, ok := emails[key]; !ok {
			add = append(add, contact)
			if !dryRun {
				err := h.GroupsClient.AddMember(contact)
				if err != nil {
					c.Logger().Errorf("Failed to add %s to members group: %s", contact, err)
					errored = append(errored, contact)
					continue
				}
				c.Logger().Infof("Added %s to members group", contact)
			}
		}
	}

	// iterate over emails to find deletions
	for key, member := range emails {
		if _, ok := contacts[key]; !ok {
			del = append(del, member)
			if !dryRun {
				err := h.GroupsClient.RemoveMember(member)
				if err != nil {
					c.Logger().Errorf("Failed to remove %s from members group: %s", member, err)
					errored = append(errored, member)
					continue
				}
				c.Logger().Infof("Removed %s from members group", member)
			}
		}
	}
	report.Groups["members"] = reconcile.Changes{
		Additions: add,
		Deletions: del,
		Errored:   errored,
	}
	if len(errored) > 0 {
		respStatus = http.StatusMultiStatus
	}

	// DISCORD
	discAdd := make(map[string][]string)
	discDel := make(map[string][]string)
	discErrored := make(map[string][]string)
	guildMembers, err := h.DiscordClient.GuildMembers()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to retrieve discord guild members", err)
	}
	contactsByDiscord := discordContactMap(contactList)
	for guild, members := range guildMembers {
		discAdd[guild] = []string{}
		discDel[guild] = []string{}
		discErrored[guild] = []string{}

		// Handle additions
		for key := range contactsByDiscord {
			m, ok := members[key]
			if !ok || h.DiscordClient.HasMemberRole(m, guild) {
				continue
			}
			discAdd[guild] = append(discAdd[guild], m.Nick())
			if !dryRun {
				err := h.DiscordClient.AddMemberRole(m.ID, guild)
				if err != nil {
					c.Logger().Errorf("Failed to add %s to %s discord member role: %s", m.Nick(), guild, err)
					discErrored[guild] = append(discErrored[guild], m.Nick())
					continue
				}
				c.Logger().Infof("Added %s to %s discord member role", m.Nick(), guild)
			}

		}
		// Handle deletions
		for _, m := range members {
			_, ok := contactsByDiscord[m.ID]
			if ok || !h.DiscordClient.HasMemberRole(m, guild) {
				continue
			}
			discDel[guild] = append(discDel[guild], m.Nick())
			if !dryRun {
				err = h.DiscordClient.RemoveMemberRole(m.ID, guild)
				if err != nil {
					c.Logger().Errorf("Failed to remove %s from %s discord member role: %s", m.Nick(), guild, err)
					errored = append(errored, m.Nick())
					continue
				}
				c.Logger().Infof("Removed %s from %s discord member role", m.Nick(), guild)
			}
		}
	}
	for g := range h.DiscordClient.Guilds {
		report.Discord[g] = reconcile.Changes{
			Additions: discAdd[g],
			Deletions: discDel[g],
			Errored:   discErrored[g],
		}
		if len(discErrored[g]) > 0 {
			respStatus = http.StatusMultiStatus
		}
	}

	// set duration
	report.Duration = time.Since(report.Date)

	// send report if changes were made
	if !dryRun && report.HasChanges() {
		err = h.EmailClient.SendReconcileReport(report)
		if err != nil {
			c.Logger().Warnf("Failed to send reconciliation report: %s", err)
		}
	}

	return c.JSON(respStatus, report)
}

func (h *Handlers) addExceptions(emails map[string]string) {
	for _, e := range h.GroupExceptions {
		emails[groupKey(e)] = e
	}
}

func contactEmailMap(slice []sfdc.Contact) map[string]string {
	contacts := make(map[string]string, len(slice))
	for _, c := range slice {
		if len(c.GroupEmail) > 0 {
			contacts[groupKey(c.GroupEmail)] = c.GroupEmail
		}
		if len(c.GroupEmailAlt) > 0 {
			contacts[groupKey(c.GroupEmailAlt)] = c.GroupEmailAlt
		}
	}
	return contacts
}

// Gmail likes to "fix" missing dots and capitalization, so we'll normalize them to prevent trying to add duplicates
func groupKey(email string) string {
	key := strings.ToLower(email)
	return strings.ReplaceAll(key, ".", "")
}

func groupEmailMap(slice []*admin.Member) map[string]string {
	members := make(map[string]string, len(slice))
	for _, m := range slice {
		members[groupKey(m.Email)] = m.Email
	}
	return members
}

func discordContactMap(slice []sfdc.Contact) map[string]sfdc.Contact {
	contacts := make(map[string]sfdc.Contact)
	for _, c := range slice {
		if len(c.DiscordID) > 0 {
			contacts[c.DiscordID] = c
		}
	}
	return contacts
}
