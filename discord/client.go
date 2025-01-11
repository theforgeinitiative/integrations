package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Client struct {
	BotSession *discordgo.Session
	Guilds     map[string]Guild
}

type Guild struct {
	ID                string `mapstructure:"id"`
	MemberRoleID      string `mapstructure:"memberRole"`
	WelcomeChannelID  string `mapstructure:"welcomeChannel"`
	DoorbellChannelID string `mapstructure:"doorbellChannel"`
}

type Member struct {
	ID         string
	ServerNick string
	Username   string
	GlobalName string
	GuildID    string
	Roles      []string
}

func (m Member) Nick() string {
	if len(m.ServerNick) > 0 {
		return m.ServerNick
	}
	if len(m.GlobalName) > 0 {
		return m.GlobalName
	}
	return m.Username
}

func NewClient(token string) (*Client, error) {
	discordBot, err := discordgo.New("Bot " + token)
	return &Client{
		BotSession: discordBot,
	}, err
}

func (c *Client) GuildMembers() (map[string]map[string]Member, error) {
	members := make(map[string]map[string]Member)
	for name, guild := range c.Guilds {
		members[name] = make(map[string]Member)
		// TODO: actually paginate this if we have over 1000 members
		guildMembers, err := c.BotSession.GuildMembers(guild.ID, "", 1000)
		if err != nil {
			return nil, fmt.Errorf("failed to get guild members for %s: %s", name, err)
		}
		for _, gm := range guildMembers {
			members[name][gm.User.ID] = Member{
				ID:         gm.User.ID,
				ServerNick: gm.Nick,
				Username:   gm.User.Username,
				GlobalName: gm.User.GlobalName,
				GuildID:    gm.GuildID,
				Roles:      gm.Roles,
			}
		}
	}

	return members, nil
}

func (c *Client) AddMemberRole(userID, guildName string) error {
	guild, ok := c.Guilds[guildName]
	if !ok {
		return fmt.Errorf("guild name %s not configured", guildName)
	}
	err := c.BotSession.GuildMemberRoleAdd(guild.ID, userID, guild.MemberRoleID)
	if err != nil {
		return fmt.Errorf("failed to add user %s to guild %s member role: %w", userID, guildName, err)
	}

	return nil
}

func (c *Client) RemoveMemberRole(userID, guildName string) error {
	guild, ok := c.Guilds[guildName]
	if !ok {
		return fmt.Errorf("guild name %s not configured", guildName)
	}
	err := c.BotSession.GuildMemberRoleRemove(guild.ID, userID, guild.MemberRoleID)
	if err != nil {
		return fmt.Errorf("failed to remove user %s from guild %s member role: %w", userID, guildName, err)
	}

	return nil
}

func (c *Client) HasMemberRole(member Member, guildName string) bool {
	guild, ok := c.Guilds[guildName]
	if !ok {
		return false
	}
	if len(guild.MemberRoleID) == 0 {
		return false
	}
	for _, r := range member.Roles {
		if r == guild.MemberRoleID {
			return true
		}
	}
	return false
}
