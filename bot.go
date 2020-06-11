package main

import (
	"container/list"
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"strings"
)

type Joke struct {
	JOKE struct {
		ID int
		JOKE string
		AUTHOR string
		LIKES int
	}
}

type Param struct {
	*list.Element
	NAME string
	VALUE string
}

func getToken() (token string) {
    data, err := ioutil.ReadFile("./config.json")
    if err != nil {
      fmt.Print(err)
    }

    type Token struct {
        TOKEN string
    }
    var obj Token

    err = json.Unmarshal(data, &obj)
    if err != nil {
        fmt.Println("error:", err)
    }

    return obj.TOKEN
}

func getJoke(s *discordgo.Session, m *discordgo.MessageCreate, url string, params []Param) Joke {
	var joke Joke
	req, err := http.NewRequest("get", url, nil)
	if err != nil {
		fmt.Println("API niet beschikbaar, ", err)
		s.ChannelMessageSend(m.ChannelID, "API niet beschikbaar.")
		return joke
	}
	q := req.URL.Query()
	for _, v := range params {
		q.Add(v.NAME, v.VALUE)
	}
	req.URL.RawQuery = q.Encode()
	resp, err :=  http.Get(req.URL.String())
	if err != nil {
		fmt.Println("API niet beschikbaar, ", err)
		s.ChannelMessageSend(m.ChannelID, "API niet beschikbaar.")
		return joke
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("API antwoord lezen mislukt, ", err)
		s.ChannelMessageSend(m.ChannelID, "API antwoord lezen mislukt.")
		return joke
	}

	err = json.Unmarshal(data, &joke)
	if err != nil {
		fmt.Println("JSON error, ", err)
		s.ChannelMessageSend(m.ChannelID, "API antwoord lezen mislukt.")
		return joke
	}
	return joke
}

func main() {
	discord, err := discordgo.New("Bot " + getToken())
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	discord.AddHandler(messageCreate)

	err = discord.Open()
	if err != nil {
		fmt.Println("Error,", err)
		return
	}

	fmt.Println("MoppenBot is gestart. Doe CTRL+C om te sluiten.")
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
    <-sc

    discord.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
        return
    }

    if strings.HasPrefix(m.Content, "!mop") {
    	args := strings.Split(m.Content, " ")[1:]
    	var param Param
		param.NAME = "likes"
		param.VALUE = "true"
    	params := []Param{param}

		if len(args) > 0 {
			if args[0] == "nsfw" {
				var param Param
				param.NAME = "nsfw"
				param.VALUE = "true"
				params = append(params, param)
			}
		}
		joke := getJoke(s, m, "https://moppenbot.nl/api/random/", params)
		if joke.JOKE.JOKE == "" {
			fmt.Println("Request error")
			s.ChannelMessageSend(m.ChannelID, "Error tijdens opvragen.")
		}
		embed := &discordgo.MessageEmbed{
			Footer:      &discordgo.MessageEmbedFooter{"Van " + joke.JOKE.AUTHOR + " | " + strconv.Itoa(joke.JOKE.LIKES) + "  👍", "", ""},
			Color:       0xffff00,
			Description: joke.JOKE.JOKE,
			Timestamp: time.Now().Format(time.RFC3339),
			Title:     "Mop " + strconv.Itoa(joke.JOKE.ID),
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
    }
}