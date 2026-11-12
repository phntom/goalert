package main

import (
	"log/slog"
	"os"
	_ "time/tzdata"

	"github.com/phntom/goalert/internal/bot"
	"github.com/phntom/goalert/internal/logging"
	"github.com/phntom/goalert/internal/sources"
)

func main() {
	logging.Setup(os.Getenv("LOG_LEVEL"))

	for _, env := range []string{"CHAT_DOMAIN", "AUTH_TOKEN"} {
		if os.Getenv(env) == "" {
			slog.Error("required environment variable is empty", "variable", env)
			os.Exit(8)
		}
	}

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
