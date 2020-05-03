//Main go file.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/session"
)

var bot *session.Session
var totalGuilds int

//The steps that main takes to start are
//1. Connects to discord and set's up bot.
//2. Adds the handlers for discord
func main() {
	//Set's up the connection to discords api.
	session, err := session.New("Bot " + os.Getenv("BOT_TOKEN"))
	//Identifies intents to receive
	//1 is guilds
	//256 is guild precences
	//512 is guild messages
	//4096 is dms
	session.Gateway.Identifier.Intents = 256 + 1 + 512 + 4096
	bot = session
	if err != nil {
		panic(err)
	}
	err = session.Open()
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.Gateway.UpdateStatus(gateway.UpdateStatusData{
		Game: &discord.Activity{
			Name: "@ to get stats",
		},
	})
	//Switched between normal status and status displaying tracked servers
	statusUpdate := time.NewTicker(time.Second * 10)
	flip := false
	go func() {
		for {
			select {
			case <-statusUpdate.C:
				var playingStr string
				if flip {
					playingStr = "Tracking stats for " + strconv.Itoa(totalGuilds) + " servers!"
					flip = false
				} else {
					playingStr = "@ to get stats"
					flip = true
				}
				session.Gateway.UpdateStatus(gateway.UpdateStatusData{
					Game: &discord.Activity{
						Name: playingStr,
					},
				})
			}
		}
	}()

	//Adds handalers for bot
	session.AddHandler(presenceUpdate)
	session.AddHandler(guildAdded)
	session.AddHandler(newMessage)

	fmt.Println("Bot is started :D")

	//Waits for the program to get a signal to close
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-exitChan
}

func newMessage(m *gateway.MessageCreateEvent) {
	if m.Author.Bot {
		return
	}

	if m.GuildID == 0 {
		//Two cases, if settings in in progress than hand it to the settings menu and if it isn't send initial message and start menu
		if _, ok := menus[m.Author.ID.String()]; ok {
			if menus[m.Author.ID.String()].settingChange == "" {
				msg := menus[m.Author.ID.String()].handleMsgInit(m)
				bot.SendMessage(m.ChannelID, msg, nil)
			} else {
				msg := menus[m.Author.ID.String()].handleMsgSetting(m)
				bot.SendMessage(m.ChannelID, msg, nil)
			}
		} else {
			msg := startMenu(m)
			bot.SendMessageComplex(m.ChannelID, api.SendMessageData{Embed: msg})
		}
	} else {
		fmt.Println("Not DM")
	}
}

//Called when guild is created, used to track how many guilds the bot is in
func guildAdded(g *gateway.GuildCreateEvent) {
	totalGuilds++
}

//Handles presence update
func presenceUpdate(p *gateway.PresenceUpdateEvent) {
	//Inital checks to weed out bad data
	activities := p.Activities
	userID := p.User.ID.String()
	if p.User.Bot {
		return
	}

	//Adds user to the container if does not exist
	if users.exists(userID) != true {
		users.add(userID)
	}

	//Goes through the activities to find new games to start tracking
	for i := range activities {
		if activities[i].Name != "Custom Status" {
			if !users.get(userID).gameExists(activities[i].Name) {
				users.get(userID).startPlaying(activities[i].Name)
			}
		}
	}

	//Checks to see if games are missing and therefore stopped playing
	for name := range users.get(userID).currentGames {
		var found bool
		for i := range activities {
			if activities[i].Name == name {
				found = true
			}
		}
		//Only called if the game is not found in the activities
		if !found {
			users.get(userID).stopPlaying(name)
		}
	}
}
