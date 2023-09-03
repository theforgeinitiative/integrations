package api

import (
	"github.com/gorilla/sessions"
	"github.com/theforgeinitiative/integrations/db"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/sfdc"
)

var sessionOpts = sessions.Options{
	Path:     "/",
	MaxAge:   86400 * 1,
	HttpOnly: true,
}

type Handlers struct {
	SFClient      *sfdc.Client
	DiscordClient *discord.Client
	DBClient      *db.Client
}
