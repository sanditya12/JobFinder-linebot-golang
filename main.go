package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/sanditya12/jobNow-linebot/scrapper"
)

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", rootHandler).Methods("GET")
	r.HandleFunc("/webhook", eventHandler).Methods("POST")

	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	data := "Hello"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	bot, err := linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_TOKEN"))
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	events, err := bot.ParseRequest(r)

	var messages []linebot.SendingMessage
	var carouselColumns []*linebot.CarouselColumn
	for _, event := range events {
		replyToken := event.ReplyToken
		if event.Type == linebot.EventTypeFollow || event.Type == linebot.EventTypeJoin {
			welcomeMessage := linebot.NewTextMessage("Welcome to JobNow!\nTo find jobs, please type the keyword for the desired job\n(e.g. Designer, Python, Intern)")
			messages = append(messages, welcomeMessage)
			_, err := bot.ReplyMessage(replyToken, messages...).Do()
			if err != nil {
				log.Println(err)
			}
		}
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				key := scrapper.CleanKey(message.Text)
				contents := scrapper.Scrap(key)
				if len(contents) == 0 {
					notFoundStr := fmt.Sprintf("Sorry, Cannot Find Any Jobs with The Keyword %q", message.Text)
					notFoundMessage := linebot.NewTextMessage(notFoundStr)
					messages = append(messages, notFoundMessage)

				} else {
					for i, content := range contents {
						if i < 5 {
							applyBtn := linebot.NewURIAction("Apply Now!", content.GetId())
							newColumn := linebot.NewCarouselColumn("", content.GetTitle(), content.GetLocation(), applyBtn)
							carouselColumns = append(carouselColumns, newColumn)
						}
					}

					resultHeader := fmt.Sprintf("Here Are The Jobs Found for The Keyword %q", message.Text)
					newMessage := linebot.NewTextMessage(resultHeader)
					messages = append(messages, newMessage)

					newCarousel := linebot.NewCarouselTemplate(carouselColumns...)
					newTemplateMessage := linebot.NewTemplateMessage(resultHeader, newCarousel)
					messages = append(messages, newTemplateMessage)
				}
				_, err := bot.ReplyMessage(replyToken, messages...).Do()
				if err != nil {
					log.Println(err)
				}
			}

		}
	}
}
