package main

import (
	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/sources"
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
	go ynet.Run()
	oref := sources.SourceOref{
		Bot: &b,
	}
	oref.Register()
	go oref.Run()

	b.AwaitMessage()
}
