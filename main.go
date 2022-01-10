package main

import (
	"context"
	"encoding/json"
	"fmt"
	sender "github.com/arikawa-req/directmessage"
	events "github.com/arikawa-req/modules"
	"github.com/arikawa-req/utilities"
	discordgo "github.com/courtier/kolizey"
	logger "github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/liamg/tml"
	"github.com/mbndr/figlet4go"
	"github.com/muesli/cache2go"
	sequences "github.com/nine-lives-later/go-windows-terminal-sequences"
	mongo "github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
	"time"
)

type jsonResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type UserInfo struct {
	userID string `bson:"id"`
}

func main() {

	sequences.EnableVirtualTerminalProcessing(syscall.Stdout, true)

	ascii := figlet4go.NewAsciiRender()

	options := figlet4go.NewRenderOptions()

	// The underscore would be an error
	renderStr, _ := ascii.RenderOpts("Noelle", options)
	tml.Println(tml.Sprintf("<blue>%s</blue>", renderStr))

	// Check for integrity
	//events.CheckIntegrity()

	// Laad configs
	godotenv.Load("config.env")

	ratelimit, _ := strconv.Atoi(os.Getenv("RATELIMIT"))
	cooldown, _ := strconv.Atoi(os.Getenv("COOLDOWN"))
	mongoURL := os.Getenv("MONGOURL")

	// Start selfbot instance
	arikawa, err := discordgo.New(os.Args[1])

	if err != nil {
		fmt.Println(err)
	}

	b, err := ioutil.ReadFile("message.txt") // just pass the file name

	if err != nil {
		fmt.Print(err)
	}

	mensagem := string(b)

	err = arikawa.Open()
	var alreadySent = cache2go.Cache("SENT")

	if err != nil {
		tml.Println("<red>[FAIL]</red>: A token caiu, o selfbot foi finalizado")
	}

	tml.Println("<green>[CONNECTED]</green>: Conta conectada com sucesso")
	// Database stuff
	ctx := context.Background()

	database, err := mongo.Open(ctx, &mongo.Config{Uri: mongoURL, Database: "arikawa", Coll: "sent"})

	if err != nil {
		tml.Println("<red>[DATABASE ERROR]</red>: Sua MongoDB não é válida ou está bode! 🐐")
		tml.Println("	<red>Aperte qualquer tecla para sair</red>")
		fmt.Scanln()
		os.Exit(3)
		return
	}

	// Add handles to interact with events
	arikawa.AddHandler(events.MessageCreate)    // Message event
	arikawa.AddHandler(events.VoiceStateUpdate) // Voice state event

	// Starting our cache-loop
	for true {

		// Blocking the program's flux trough non-active loop
		for len(events.UserList) <= 0 {
		}

		for member := 0; member < len(events.UserList); member++ {

			selected := events.UserList[member]

			if alreadySent.Exists(selected.Id) {
				continue
			}

			dbUserFound := UserInfo{}
			database.Find(ctx, bson.M{"userID": selected.Id}).One(&dbUserFound)

			if len(dbUserFound.userID) > 0 {
				tml.Printf("<lightred>[FAIL 📦]</lightred>: O membro %s já está na DB\n", selected.Tag)
				alreadySent.Add(selected.Id, 0, "SENT")
				continue
			}

			channel, _ := sender.OpenChannel(os.Args[1], selected.Id)
			resp, err := sender.SendMessage(os.Args[1], channel, &utilities.Message{Content: mensagem}, selected.Id)

			if err != nil {
				tml.Println("<lightred>[CRITICAL ERROR]</lightred>: Erro ao enviar mensagem!")
			}

			// Response body
			body, err := utilities.ReadBody(*resp)

			if err != nil {
				logger.Red("<lightred>[CRITICAL ERROR]</lightred> Erro ao processar request da mensagem!")
			}

			var response jsonResponse
			errx := json.Unmarshal(body, &response)

			if errx != nil {
				tml.Println("<lightred>[UNKNOWN ERROR]</lightred> Erro desconhecido, contacte o vendedor seu selfbot ou o nosso supote!")
				continue
			}

			if resp.StatusCode == 200 {

				tml.Printf("<green>[SENT]</green>: Mensagem enviada com sucesso para %s\n", selected.Tag)
				tml.Printf("<blue>[CD]</blue>: Esperando %d segundos para continuar\n", cooldown)

				alreadySent.Add(selected.Id, 0, "a")
				database.InsertOne(ctx, UserInfo{userID: selected.Id})
				time.Sleep(time.Duration(cooldown) * time.Second)
				continue

			} else if resp.StatusCode == 403 && response.Code == 40003 {
				tml.Printf("<red>[RATELIMIT]</red>: Ratelimit detectado, aguardando %d minutos para continuar\n", ratelimit)
				time.Sleep(time.Duration(ratelimit) * time.Minute)
			} else if resp.StatusCode == 403 && response.Code == 50007 {
				tml.Printf("<blue>[LOCKED]</blue>: Privado trancado, pulando para o próximo membro\n")
				alreadySent.Add(selected.Id, 0, "a")
				continue
			} else if (resp.StatusCode == 403 && response.Code == 40002) || resp.StatusCode == 401 || resp.StatusCode == 405 {

				if resp.StatusCode == 40002 {
					deadTokenDialogue()
					return
				}

				return
			} else if resp.StatusCode == 403 && response.Code == 50009 {
				tml.Println("<red>[FAIL]</red>: Joiner bode, não aceitou as regras do servidor.")
				continue
			}
		}
	}

	<-make(chan struct{}) // Keep the program open
	defer arikawa.Close() // Safely close the Discord's connection
}

func deadTokenDialogue() {
	tml.Println("<red>[DEAD]</red>: A token morreu, pressione qualquer tecla para continuar")
	fmt.Scanln()
	os.Exit(3)
}
