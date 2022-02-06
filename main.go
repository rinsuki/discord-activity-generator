package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

func main() {
	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	discord.StateEnabled = true
	if err != nil {
		panic(err)
	}

	discord.AddHandler(interactionHandler)

	if err := discord.Open(); err != nil {
		panic(err)
	}

	createCommandIfNeeded(discord)

	fmt.Println("Running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	discord.Close()
}

// TODO: check this function working intended
func createCommandIfNeeded(discord *discordgo.Session) {
	version := "1.1.0"
	versionString := " (v." + version + ")"

	commands, err := discord.ApplicationCommands(discord.State.User.ID, "")
	if err != nil {
		panic(err)
	}
	for _, command := range commands {
		if strings.HasSuffix(command.Description, versionString) {
			return
		}
		if err := discord.ApplicationCommandDelete(discord.State.User.ID, "", command.ID); err != nil {
			panic(err)
		}
	}
	if _, err := discord.ApplicationCommandCreate(discord.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "activity",
		Description: "Create Link to Launch Specified Activity in currently joined voice channel" + versionString,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "activity",
				Description: "Type of Activity",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					// from https://github.com/DjDeveloperr/ActivitiesBot/blob/52fab01de173de1d164f370486b9d094e5c5a22f/mod.ts
					{
						Name:  "YouTube Together (Old)",
						Value: "755600276941176913",
					},
					{
						Name: "Watch Together (new version of YouTube Together)",
						Value: "880218394199220334",
					},
					{
						Name:  "Poker Night",
						Value: "755827207812677713",
					},
					{
						Name:  "Betrayal.io",
						Value: "773336526917861400",
					},
					{
						Name:  "Fishington.io",
						Value: "814288819477020702",
					},
					{
						Name: "88055924547140816 (???)",
						Value: "88055924547140816",
					},
					{
						Name: "Sketch Heads",
						Value: "902271654783242291",
					},
				},
			},
		},
	}); err != nil {
		panic(err)
	}
}

func interactionHandler(discord *discordgo.Session, interact *discordgo.InteractionCreate) {
	var finished bool
	defer func() {
		if !finished {
			if err := discord.InteractionRespond(interact.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionApplicationCommandResponseData{
					Content: "Sorry! Failed to create link...",
					Flags:   64,
				},
			}); err != nil {
				println(err)
			}
		}
	}()

	voiceChannelState, err := discord.State.VoiceState(interact.GuildID, interact.Member.User.ID)
	if err != nil {
		println(err)
		return
	}

	if voiceChannelState == nil {
		if err := discord.InteractionRespond(interact.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionApplicationCommandResponseData{
				Content: "You should join voice channels first. (If you already joined, please leave and join)",
				Flags:   64,
			},
		}); err != nil {
			println(err)
		}
		finished = true
		return
	}
	voiceChannelID := voiceChannelState.ChannelID

	appID := interact.Interaction.Data.Options[0].StringValue()
	data := struct {
		MaxAge              int    `json:"max_age"`
		MaxUses             int    `json:"max_uses"`
		Temporary           bool   `json:"temporary"`
		Unique              bool   `json:"unique"`
		TargetType          int    `json:"target_type"`
		TargetApplicationID string `json:"target_application_id"`
	}{
		MaxAge:              10,
		MaxUses:             1,
		Temporary:           true,
		Unique:              true,
		TargetType:          2,
		TargetApplicationID: appID,
	}

	body, err := discord.RequestWithBucketID("POST", discordgo.EndpointChannelInvites(voiceChannelID), data, discordgo.EndpointChannelInvites(voiceChannelID))
	if err != nil {
		println(err)
		return
	}

	var st discordgo.Invite
	if err := json.Unmarshal(body, &st); err != nil {
		println(err)
		return
	}

	if err := discord.InteractionRespond(interact.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionApplicationCommandResponseData{
			Content: "Link Created! [Click Here to Start](https://discord.gg/" + st.Code + ") (this link will expire after 10 seconds)",
			Flags:   64,
		},
	}); err != nil {
		println(err)
	}
	finished = true

	go func() {
		time.Sleep(10 * time.Second)
		if err := discord.InteractionResponseEdit(discord.State.User.ID, interact.Interaction, &discordgo.WebhookEdit{
			Content: "Link Created! (Link has been expired)",
		}); err != nil {
			println(err)
		}
	}()
}
