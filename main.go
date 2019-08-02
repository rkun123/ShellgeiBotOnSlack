// +build !test

package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/mattn/go-sixel"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nlopes/slack"
)

func processMessage(msg slack.Msg, self slack.UserDetails, api *slack.Client, db *sql.DB, config botConfig) {
	fmt.Printf("msg.User: %s, msg.BotID: %s, self.ID: %s, msg.Text: %s\n", msg.User, msg.BotID, self.ID, msg.Text)
	// check if it is valid shellgei tweet
	if msg.User == "" {
		return
	}

	// is mentioned?
	if !strings.Contains(msg.Text, fmt.Sprintf("<@%s>", self.ID)) {
		return
	}

	text, mediaUrls, err := extractShellgei(msg, self, api)
	fmt.Printf("text: %s\n", text)
	if err != nil {
		log.Println(err)
		return
	}

	//insertShellGei(db, msg.User, msg.Username, msg.Timestanp, text, t.Unix())

	result, b64imgs, err := runCmd(text, mediaUrls, config)
	//insertResult(db, msg.Timestamp, result, err)
	fmt.Printf("result: %s\n", result)

	if err != nil {
		_, _, _ = api.PostMessage(msg.Channel, slack.MsgOptionText("internal error", true))
		fmt.Println(err)
		return
	}

	if len(result) == 0 && len(b64imgs) == 0 {
		return
	}

	err = postResult(api, msg, result, b64imgs)
	if err != nil {
		log.Println(err)
	}
	return
}

/// ShellgeiBot main function
func botMain(slackConfigFile, botConfigFile string) {
	token, err := parseSlackKey(slackConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}
	_, _ = db.Exec(schema)

	api := slack.New(token)

	config, err := parseBotConfig(botConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	self := slack.UserDetails{}
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		fmt.Println("Event Received!!")
		switch ev := msg.Data.(type) {

		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			self = *ev.Info.User
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			config, err = parseBotConfig(botConfigFile)
			if err != nil {
				rtm.SendMessage(rtm.NewOutgoingMessage("Internal Error", ev.Channel))
				log.Fatal(err)
			}

			go func() {
				messageEvent := msg.Data.(*slack.MessageEvent)

				message := slack.Msg{
					Text:      messageEvent.Text,
					User:      messageEvent.User,
					Channel:   messageEvent.Channel,
					Timestamp: messageEvent.Timestamp,
				}
				processMessage(message, self, api, db, config)
			}()
		}
	}
}

func botTest(botConfigFile, scriptFile string) {
	config, err := parseBotConfig(botConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	script, err := ioutil.ReadFile(scriptFile)
	if err != nil {
		log.Fatal(err)
	}

	result, b64imgs, err := runCmd(string(script), []string{}, config)

	if err != nil {
		if err.(*stdError) == nil {
			log.Fatal("internal Error")
		}
		return
	}

	if len(result) == 0 && len(b64imgs) == 0 {
		fmt.Println("No result")
		return
	}

	fmt.Println(result)
	fmt.Println(len(b64imgs))
	for _, b64img := range b64imgs {
		imgBytes, err := base64.StdEncoding.DecodeString(b64img)
		if err != nil {
			log.Println(err)
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			log.Println(err)
			continue
		}

		sixel.NewEncoder(os.Stdout).Encode(img)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("<Usage>%s: TwitterConfig.json ShellgeiConfig.json | -test ShellgeiConfig.json script", os.Args[0])
	}

	if os.Args[1] == "-test" {
		// testing mode
		botTest(os.Args[2], os.Args[3])
	} else {
		// normal mode
		botMain(os.Args[1], os.Args[2])
	}
}
