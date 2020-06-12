package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Joke struct {
	JOKE struct {
		ID     int
		JOKE   string
		AUTHOR string
		LIKES  int
	}
}

type Param struct {
	*list.Element
	NAME  string
	VALUE string
}

var apiKey string

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

func setApiKey() {
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Print(err)
	}

	type ApiKey struct {
		API_KEY string
	}
	var obj ApiKey

	err = json.Unmarshal(data, &obj)
	if err != nil {
		fmt.Println("error:", err)
	}

	apiKey = obj.API_KEY
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
	resp, err := http.Get(req.URL.String())
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

func sendLike(url string, params []Param) {
	req, err := http.NewRequest("get", url, nil)
	if err != nil {
		fmt.Println("API niet beschikbaar, ", err)
	}
	q := req.URL.Query()
	for _, v := range params {
		q.Add(v.NAME, v.VALUE)
	}
	q.Add("api_key", apiKey)
	req.URL.RawQuery = q.Encode()
	resp, err := http.Get(req.URL.String())
	if err != nil {
		fmt.Println("API niet beschikbaar, ", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
}

func setStatus(s *discordgo.Session, event *discordgo.Ready) {
	guilds := s.State.Guilds
	s.UpdateStatus(0, "!mop | "+strconv.Itoa(len(guilds))+" guilds")
	statusTimer := time.NewTimer(20 * time.Second)
	go func() {
		<-statusTimer.C
		setStatus(s, event)
	}()
}

func main() {
	setApiKey()
	discord, err := discordgo.New("Bot " + getToken())
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	discord.AddHandler(onReady)
	discord.AddHandler(messageCreate)
	discord.AddHandler(messageReactionAdd)

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

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	setStatus(s, event)
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
			} else {
				query := strings.Join(args, " ")
				param.NAME = "q"
				param.VALUE = query
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
			Timestamp:   time.Now().Format(time.RFC3339),
			Title:       "Mop " + strconv.Itoa(joke.JOKE.ID),
		}

		jokeMsg, _ := s.ChannelMessageSendEmbed(m.ChannelID, embed)

		s.MessageReactionAdd(jokeMsg.ChannelID, jokeMsg.ID, "👍")
	}
}

func messageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	user, _ := s.User(r.UserID)
	if user.Bot {
		return
	}
	if r.Emoji.Name != "👍" {
		return
	}
	msg, _ := s.ChannelMessage(r.ChannelID, r.MessageID)
	jokeID := strings.Replace(msg.Embeds[0].Title, "Mop ", "", -1)
	var jokeIDParam Param
	jokeIDParam.NAME = "joke"
	jokeIDParam.VALUE = jokeID
	var userIDParam Param
	userIDParam.NAME = "user"
	userIDParam.VALUE = r.UserID
	params := []Param{jokeIDParam, userIDParam}
	sendLike("https://moppenbot.nl/api/like/", params)
}
