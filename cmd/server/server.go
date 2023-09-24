package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"

	"github.com/theforgeinitiative/integrations/api"
	"github.com/theforgeinitiative/integrations/checkmein"
	"github.com/theforgeinitiative/integrations/config"
	"github.com/theforgeinitiative/integrations/discord"
	"github.com/theforgeinitiative/integrations/groups"
	"github.com/theforgeinitiative/integrations/mail"
	"github.com/theforgeinitiative/integrations/sfdc"
)

func main() {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.INFO)

	err := config.LoadConfig()
	if err != nil {
		e.Logger.Fatalf("Failed to load config: %s", err)
	}

	// setup SFDC connection
	sfClient, err := sfdc.NewClient(viper.GetString("sfdc.url"), viper.GetString("sfdc.clientId"), viper.GetString("sfdc.clientSecret"))
	if err != nil {
		e.Logger.Fatal("Failed to create SFDC client", err)
	}

	// // DB config
	// firestoreClient, err := db.NewClient(gcpProjectID)
	// if err != nil {
	// 	e.Logger.Fatal("Failed to create Firestore client", err)
	// }

	// setup Discord session
	discordClient, err := discord.NewClient(viper.GetString("discord.botToken"))
	if err != nil {
		e.Logger.Fatal("Failed to create Discord client", err)
	}
	err = viper.UnmarshalKey("discord.guilds", &discordClient.Guilds)
	if err != nil {
		e.Logger.Fatal("Failed to read Discord guild config", err)
	}

	// google groups client
	gc, err := groups.NewClient(viper.GetString("groups.members.email"))
	if err != nil {
		log.Fatalf("Client err: %s", err)
	}

	// checkmein client
	cc := checkmein.NewClient(viper.GetString("checkmein.url"), viper.GetString("checkmein.username"), viper.GetString("checkmein.password"))

	// email client
	mc := mail.NewClient(viper.GetString("mail.apiKey"), viper.GetString("mail.fromName"), viper.GetString("mail.fromEmail"), viper.GetString("mail.to"))

	// create handler struct
	app := api.Handlers{
		SFClient:      &sfClient,
		DiscordClient: discordClient,
		//DBClient:        firestoreClient,
		GroupsClient:    &gc,
		GroupExceptions: viper.GetStringSlice("groups.members.exceptions"),
		CheckMeInClient: &cc,
		EmailClient:     &mc,
	}

	// api routes
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "🤖🛠️😎")
	})
	e.POST("/api/v1/reconcile", app.Reconcile)

	e.Logger.Fatal(e.Start(":3000"))
}
