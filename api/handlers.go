package api

import (
	"github.com/gorilla/sessions"
	"github.com/theforgeinitiative/integrations/checkmein"
	"github.com/theforgeinitiative/integrations/db"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/groups"
	"github.com/theforgeinitiative/integrations/mail"
	"github.com/theforgeinitiative/integrations/sfdc"
)

var sessionOpts = sessions.Options{
	Path:     "/",
	MaxAge:   86400 * 1,
	HttpOnly: true,
}

type Handlers struct {
	SFClient        *sfdc.Client
	DiscordClient   *discord.Client
	DBClient        *db.Client
	GroupsClient    *groups.Client
	GroupExceptions []string
	CheckMeInClient *checkmein.Client
	EmailClient     *mail.Client
}
