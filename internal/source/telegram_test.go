package source

import (
	"context"
	"testing"
	"time"

	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/metrics"
)

const telegramAlertMsg = "ירי רקטות וטילים מתימן (15/06/2026) 08:06\nתל אביב (90 שניות)\nרמת גן (מיידי)"

func TestExtractTelegramCities(t *testing.T) {
	cities := extractTelegramCities(telegramAlertMsg)
	if len(cities) != 2 || cities[0] != "תל אביב" || cities[1] != "רמת גן" {
		t.Fatalf("cities = %v", cities)
	}
}

func TestExtractTelegramPubTime(t *testing.T) {
	if got := extractTelegramPubTime(telegramAlertMsg); got != "15/06/2026 08:06" {
		t.Errorf("pubtime = %q", got)
	}
	if got := extractTelegramPubTime("(1/2/2026) 9:05"); got != "01/02/2026 9:05" {
		t.Errorf("padding = %q", got)
	}
	if got := extractTelegramPubTime("no time here"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestTelegramCategory(t *testing.T) {
	if c, k := telegramCategory("anything", true); c != alert.CatPreAlert || k != alert.KindPreAlert {
		t.Errorf("early = %v,%v", c, k)
	}
	if c, _ := telegramCategory("חדירת כלי טיס עוין", false); c != alert.CatUAV {
		t.Errorf("uav misclassified: %v", c)
	}
	if c, _ := telegramCategory("חדירת מחבלים", false); c != alert.CatTerror {
		t.Errorf("terror misclassified: %v", c)
	}
	if c, _ := telegramCategory("ירי רקטות וטילים", false); c != alert.CatMissile {
		t.Errorf("missile misclassified: %v", c)
	}
}

func TestTelegramExpired(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	at := time.Date(2026, 6, 15, 8, 6, 0, 0, loc)
	if telegramExpired("15/06/2026 08:06", at.Add(30*time.Second), false) {
		t.Error("should not be expired within TTL")
	}
	if !telegramExpired("15/06/2026 08:06", at.Add(2*time.Minute), false) {
		t.Error("should be expired past 90s")
	}
	if telegramExpired("15/06/2026 08:06", at.Add(2*time.Minute), true) {
		t.Error("early alert TTL is 300s, should not be expired at 2m")
	}
}

func TestTelegramHandleAlertEmitsOrigin(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	at := time.Date(2026, 6, 15, 8, 6, 0, 0, loc)
	out := make(chan alert.Alert, 1)
	tg := &Telegram{out: out, metrics: metrics.New()}
	tg.handleAlert(context.Background(), telegramAlertMsg, at.Add(10*time.Second))
	select {
	case a := <-out:
		if a.Category != alert.CatMissile || a.Kind != alert.KindAlert {
			t.Errorf("category/kind = %v/%v", a.Category, a.Kind)
		}
		if a.Origin != "yemen" {
			t.Errorf("origin = %q, want yemen", a.Origin)
		}
		if len(a.Areas) != 2 {
			t.Errorf("areas = %v", a.Areas)
		}
	default:
		t.Fatal("no alert emitted")
	}
}

func TestTelegramHandleEndAlert(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	at := time.Date(2026, 6, 15, 8, 6, 0, 0, loc)
	text := "האירוע הסתיים (15/06/2026) 08:06\nתל אביב (90 שניות)"
	out := make(chan alert.Alert, 1)
	tg := &Telegram{out: out, metrics: metrics.New()}
	tg.handleAlert(context.Background(), text, at.Add(10*time.Second))
	select {
	case a := <-out:
		if a.Kind != alert.KindEnd || a.Category != alert.CatEndAlert {
			t.Errorf("expected end alert, got %v/%v", a.Kind, a.Category)
		}
	default:
		t.Fatal("no end alert emitted")
	}
}
