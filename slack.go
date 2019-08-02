package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/nlopes/slack"
)

type slackKeys struct {
	Token string `json:"Token"`
}

func parseSlackKey(file string) (string, error) {
	var tokenFile slackKeys
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(raw, &tokenFile)
	if err != nil {
		return tokenFile.Token, err
	}
	return tokenFile.Token, nil
}

func fetchMyUserInfo(bot string, api slack.Client) (*slack.Bot, error) {
	return api.GetBotInfo(bot)
}

func extractShellgei(msg slack.Msg, self slack.UserDetails, api *slack.Client) (string, []string, error) {
	text := msg.Text
	if len(text) == 0 {
		return "", []string{""}, errors.New("Message is without text")
	}

	text = strings.Replace(text, "<"+self.ID+">", "", -1)

	// Extract files and fetch public URLs
	fileURLs := make([]string, 0, 4)
	files := msg.Files
	if len(files) > 0 {
		for _, file := range files {
			sharedFile, _, _, err := api.ShareFilePublicURL(file.ID)
			if err != nil {
				break
			}
			fileURLs = append(fileURLs, sharedFile.URL)
		}

	}
	return text, fileURLs, nil
}

func postResult(api *slack.Client, msg slack.Msg, result string, b64imgs []string) error {
	for i, b64img := range b64imgs {
		params := slack.FileUploadParameters{
			File:    string(i),
			Content: b64img,
		}

		_, err := api.UploadFile(params)
		if err != nil {
			return err
		}
	}
	_, _, err := api.PostMessage(msg.Channel, slack.MsgOptionText("internal error", true))
	if err != nil {
		return err
	}
	return nil
}
