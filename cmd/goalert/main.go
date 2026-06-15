// Command goalert runs the Pikud HaOref alert bot: it ingests alerts from
// ynet, the oref history feed and the Tzeva Adom WebSocket, then publishes the
// fastest possible message to Mattermost.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/metrics"
	"github.com/phntom/goalert/internal/publish"
	"github.com/phntom/goalert/internal/source"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	m := metrics.New()
	go m.Serve(env("METRICS_ADDR", ":3000"))

	areas := area.Default()

	chat := publish.NewMattermost(os.Getenv("CHAT_DOMAIN"), os.Getenv("AUTH_TOKEN"), m)
	if err := chat.Connect(ctx); err != nil {
		mlog.Error("failed to connect to mattermost", mlog.Err(err))
		os.Exit(1)
	}
	if err := chat.FindChannels(ctx); err != nil {
		mlog.Error("failed to find channels", mlog.Err(err))
		os.Exit(1)
	}

	feed := make(chan alert.Alert, 64)
	for _, s := range buildSources(areas, m) {
		go s.Run(ctx, feed)
		mlog.Info("source started", mlog.String("source", s.Name()))
	}

	publish.NewEngine(chat, areas, m).Run(ctx, feed)
	mlog.Info("shutting down")
}

// buildSources assembles the enabled alert sources. Each can be turned off with
// its DISABLE_* env var.
func buildSources(areas *area.Set, m *metrics.Metrics) []source.Source {
	var srcs []source.Source
	if !disabled("DISABLE_YNET") {
		srcs = append(srcs, source.NewYnet(os.Getenv("YNET_URL"), m))
	}
	if !disabled("DISABLE_OREF") {
		srcs = append(srcs, source.NewOrefHistory(os.Getenv("OREF_HISTORY_URL"), m))
	}
	if !disabled("DISABLE_TZEVAADOM") {
		srcs = append(srcs, source.NewTzevaadom(os.Getenv("TZEVAADOM_WS_URL"), areas, m))
	}
	return srcs
}

func disabled(key string) bool { return os.Getenv(key) == "1" }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
