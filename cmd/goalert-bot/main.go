package main

import (
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/sources"
	"os"
)

func main() {
	b := bot.Bot{}
	b.Register()
	b.Connect()
	b.FindBotChannel()
	go b.Cleanup()

	ynet := sources.SourceYnet{
		URL: sources.YnetURL,
		Bot: &b,
	}
	ynet.Register()
	if os.Getenv("DISABLE_YNET") != "1" {
		go ynet.Run()
	}
	oref := sources.SourceOref{
		Bot: &b,
	}
	oref.Register()
	if os.Getenv("DISABLE_OREF") != "1" {
		go oref.Run()
	}
	telegram := sources.SourceTelegram{
		Bot: &b,
	}
	telegram.Register()
	if os.Getenv("DISABLE_TELEGRAM") != "1" {
		go telegram.Run()
	}
	b.AwaitMessage()
}
