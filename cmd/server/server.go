package main

import (
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"

	"github.com/theforgeinitiative/integrations/api"
	"github.com/theforgeinitiative/integrations/db"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/sfdc"
)

var (
	sfURL               = os.Getenv("SF_URL")
	clientID            = os.Getenv("SF_CLIENT_ID")
	clientSecret        = os.Getenv("SF_CLIENT_SECRET")
	discordBotToken     = os.Getenv("DISCORD_BOT_TOKEN")
	discordClientID     = os.Getenv("DISCORD_CLIENT_ID")
	discordClientSecret = os.Getenv("DISCORD_CLIENT_SECRET")
	discordRedirectURL  = os.Getenv("DISCORD_REDIRECT_URL")
	gcpProjectID        = os.Getenv("GCP_PROJECT_ID")
)

func main() {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Use(middleware.Logger())

	// setup SFDC connection
	sfClient, err := sfdc.NewClient(sfURL, clientID, clientSecret)
	if err != nil {
		e.Logger.Fatal("Failed to create SFDC client", err)
	}

	// DB config
	firestoreClient, err := db.NewClient(gcpProjectID)
	if err != nil {
		e.Logger.Fatal("Failed to create Firestore client", err)
	}

	// setup Discord session
	discordBot, err := discordgo.New("Bot " + discordBotToken)
	if err != nil {
		e.Logger.Fatal("Failed to create Discord client", err)
	}

	oauthConfig := oauth2.Config{
		ClientID:     discordClientID,
		ClientSecret: discordClientSecret,
		Scopes:       []string{"identify", "role_connections.write"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
		RedirectURL: discordRedirectURL,
	}

	// create handler struct
	app := api.Handlers{
		SFClient: &sfClient,
		DiscordClient: &discord.Client{
			BotSession:  discordBot,
			OAuthConfig: &oauthConfig,
			Store:       firestoreClient,
		},
		DBClient: firestoreClient,
	}

	// api routes
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "🤖🛠️😎")
	})
	e.GET("/discord", app.DiscordLanding)
	e.GET("/discord/redirect", app.LinkRoleRedirect)
	e.GET("/discord/callback", app.LinkRoleCallback)
	e.GET("/discord/register", app.LinkRoleRegister)
	e.GET("/discord/appcallback", app.MemberAppCallback)
	e.POST("/discord/verify", app.VerifyByIDs)

	e.Logger.Fatal(e.Start(":3000"))
}
