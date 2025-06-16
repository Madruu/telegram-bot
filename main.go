package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

//Creating the bot struct
type Bot struct {
	bot *telego.Bot
	bh *th.BotHandler
	token string
}

//Stores the sympla api responses
type SymplaResponse struct {
	Data []Event `json:"data"`
}

//Structure to store responses
type Location struct {
	Country string `json:"country"`
	Address string `json:"address"`
	AddressAlt string `json:"address_alt"`
	City string `json:"city"`
	AddressNum string `json:"address_num"`
	Name string `json:"name"`
	Longitude float64 `json:"lon"`
	State string `json:"state"`
	Neighbourhood string `json:"neighbourhood"`
	ZipCode string `json:"zip_code"`
	Latitude float64 `json:"lat"`
}

//Stores images related to an event
type Images struct {
	Original string `json:"original"`
	XS string `json:"xs"`
	LG string `json:"lg"`
}

//Stores start/end date for events
type StartDateFormats struct {
	Pt string `json:"pt"`
	En string `json:"en"`
	Es string `json:"es"`
}

type EndDateFormats struct {
	Pt string `json:"pt"`
	En string `json:"en"`
	Es string `json:"es"`
}

//Stores the events
type Event struct {
	Name string `json:"name"`
	Images string `json:"images"`
	Location string `json:"location"`
	StartDateFormats StartDateFormats `json:"start_date_formats"`
	EndDateFormats EndDateFormats `json:"end_date_formats"`
	URL string `json:"url"`
}

func NewBot(token string) (*Bot, error) {
	bot, err := telego.NewBot(token, telego.WithDefaultDebugLogger())
	if err != nil {
		return nil, err;
	}

	updates, err := bot.UpdatesViaLongPolling(nil);
	if err != nil {
		return nil, err
	}
	
	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return nil, err
	}

	return &Bot {
		bot: bot,
		bh: bh,
		token: token,
	}, nil
}

//Initializes bot
func (b *Bot) Start() {
    defer b.bh.Stop()
    defer b.bot.StopLongPolling()

    b.registerCommands()

    b.bh.Start()
}


func (b *Bot) registerCommands(){
	b.registerBotCommand();
	b.registerEventCommands();
}

func (b *Bot) registerBotCommand() {
	b.bh.Handle(func(bot *telego.Bot, update telego.Update) {
		infoMessage := `Alo`

		_, _ = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			infoMessage,
		))
	}, th.CommandEqual("start"))
}

func (b *Bot) registerEventCommands() {
	b.registerAvailableEventCommand();
	b.registerClosedEventCommand();
}

func (b *Bot) registerAvailableEventCommand() {
	b.bh.Handle(func(bot *telego.Bot, udpate telego.Update) {
		events, err := fetchSymplaEvents("future")
		if err != nil {
			fmt.Println("Erro ao buscar eventos: ", err)
			return
		}

		message := formatEventsMessage(events)
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			message,
		))
	}, th.CommandEqual("disponíveis"))
}

func (b *Bot) registerClosedEventCommand() {
    b.bh.Handle(func(bot *telego.Bot, update telego.Update) {
        events, err := fetchSymplaEvents("past")
        if err != nil {
            fmt.Println("Erro ao buscar eventos:", err)
            return
        }
        message := formatEventsMessage(events)
        _, _ = bot.SendMessage(tu.Message(
            tu.ID(update.Message.Chat.ID),
            message,
        ))
    }, th.CommandEqual("encerrados"))
}

func fetchSymplaEvents(eventType string) ([]Event, error) {
	//Generates organizer's ids
	organizerIds := []int{3125215, 5478152}

	//Defines the service to be called by the Sympla API
	service := "/v4/search"
	if eventType == "past" {
		service = "/v4/events/past"
	}

	//Assembles the body request
	requestBody := fmt.Sprintf(`{
        "service": "%s",
        "params": {
            "only": "name,images,location,start_date_formats,end_date_formats,url",
            "organizer_id": %s,
            "sort": "date",
            "order_by": "desc",
            "limit": "6",
            "page": 1
        },
        "ignoreLocation": true
    }`, service, intArrayToString(organizerIds))

	//Makes HTTP request to Sympla's API
	resp, err := http.Post("https://www.sympla.com.br/api/v1/search", "application/json", strings.NewReader(requestBody))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close();

	//Reads response's request
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//Decodes JSON response in a SymplaResponse structure
	var symplaResp SymplaResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&symplaResp); err != nil {
		return nil, err
	}

	//Returns the events
	return symplaResp.Data, nil
	
}

func intArrayToString(arr []int) string {
	strArr := make([]string, len(arr))
	for i, num := range arr {
		strArr[i] = fmt.Sprint(num)
	}
	return "[" + strings. Join(strArr, ",") + "[]"
}

func formatEventsMessage(events []Event) string {
	message := "#BOT DEVS OF LATIN"
	if events == nil || len(events) == 0 {
		message += "Ops... Looks Like the service is unavailable at the moment :("
	} else {
		message += "Events:"
		for _, event := range events {
			message += fmt.Sprintf("- %s\n  Local: %s\n  Data: %s\n  URL: %s\n \n\n\n", event.Name, event.Location.City, event.StartDateFormats.Pt, event.URL)
            message += "----------------------------------------\n\n\n"
		}
	}

	return message
}


func main() {
	//Loads the environment variables from the .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Erro ao carregar o arquivo .env: ", err)
		os.Exit(1)
	}

	//Obtains the token of the telegram bot from the env variables
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		fmt.Println("Bots token not found")
		os.Exit(1)
	}


	//Creates a new bot instance and initiates its execution
	bot, err := NewBot(token)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	bot.Start();
}
