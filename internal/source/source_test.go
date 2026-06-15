package source

import (
	"fmt"
	"testing"
	"time"

	"github.com/phntom/goalert/internal/alert"
	"github.com/phntom/goalert/internal/area"
	"github.com/phntom/goalert/internal/i18n"
)

func ynetBody(items string) []byte {
	return []byte(fmt.Sprintf(`jsonCallback({"alerts": {"items": [%s]}});`, items))
}

func ynetItem(guid, title, desc string) string {
	return fmt.Sprintf(`{"item":{"guid":%q,"pubdate":"11:40","title":%q,"description":%q}}`, guid, title, desc)
}

const ynetRocket = "היכנסו למרחב המוגן ושהו בו 10 דקות"

func TestYnetParseGroupsAndDedups(t *testing.T) {
	y := &Ynet{seen: map[string]bool{}}
	body := ynetBody(ynetItem("g1", "בית שקמה", ynetRocket) + "," + ynetItem("g2", "אשקלון", ynetRocket))
	got := y.parse(body)
	if len(got) != 1 {
		t.Fatalf("want 1 grouped alert, got %d", len(got))
	}
	if got[0].Category != alert.CatMissile || len(got[0].Areas) != 2 || len(got[0].IDs) != 2 {
		t.Errorf("unexpected alert: %+v", got[0])
	}
	if again := y.parse(body); len(again) != 0 {
		t.Errorf("seen guids should suppress re-emit, got %d", len(again))
	}
}

func TestYnetParseDrillAndUnknownSkipped(t *testing.T) {
	y := &Ynet{seen: map[string]bool{}}
	body := ynetBody(
		ynetItem("d1", "עיר", "אזעקה במסגרת תרגיל") + "," +
			ynetItem("u1", "עיר", "טקסט לא ידוע"))
	if got := y.parse(body); len(got) != 0 {
		t.Errorf("drill+unknown should yield no alerts, got %d", len(got))
	}
}

func TestYnetEmptyClearsSeen(t *testing.T) {
	y := &Ynet{seen: map[string]bool{"old": true}}
	if got := y.parse(ynetBody("")); got != nil {
		t.Errorf("empty feed should return nil, got %v", got)
	}
	if len(y.seen) != 0 {
		t.Errorf("empty feed should clear seen, have %d", len(y.seen))
	}
}

func orefDate(at time.Time) string {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	return at.In(loc).Format("2006-01-02T15:04:05")
}

func TestOrefPrimeThenEmit(t *testing.T) {
	o := &OrefHistory{seen: map[int]bool{}}
	now := time.Now()
	body1 := []byte(fmt.Sprintf(`[{"data":"חיפה","alertDate":%q,"category":1,"rid":100}]`, orefDate(now)))
	if got := o.parse(body1, now); len(got) != 0 {
		t.Fatalf("first poll must prime only, got %d", len(got))
	}
	body2 := []byte(fmt.Sprintf(
		`[{"data":"נהריה","alertDate":%q,"category":13,"rid":101},{"data":"חיפה","alertDate":%q,"category":1,"rid":100}]`,
		orefDate(now), orefDate(now)))
	got := o.parse(body2, now)
	if len(got) != 1 {
		t.Fatalf("want 1 new alert, got %d", len(got))
	}
	if got[0].Kind != alert.KindEnd || got[0].Category != alert.CatEndAlert {
		t.Errorf("category 13 should be an end alert: %+v", got[0])
	}
}

func TestOrefRecencyFilter(t *testing.T) {
	o := &OrefHistory{seen: map[int]bool{}, primed: true}
	now := time.Now()
	old := []byte(fmt.Sprintf(`[{"data":"חיפה","alertDate":%q,"category":1,"rid":200}]`, orefDate(now.Add(-time.Hour))))
	if got := o.parse(old, now); len(got) != 0 {
		t.Errorf("stale record should be filtered, got %d", len(got))
	}
}

func tzevaTestSet() *area.Set {
	return area.NewSet([]*area.Area{{ID: "1", TzevaID: 511, Names: map[i18n.Language]string{"he": "אבו גוש"}}})
}

func TestTzevaadomAlert(t *testing.T) {
	tz := &Tzevaadom{seen: map[string]bool{}, areas: tzevaTestSet()}
	msg := []byte(`{"type":"ALERT","data":{"notificationId":"n1","time":1751284800,"threat":5,"isDrill":false,"cities":["גבעת שמואל"]}}`)
	a, ok := tz.parseMessage(msg)
	if !ok || a.Category != alert.CatUAV || a.Kind != alert.KindAlert || len(a.Areas) != 1 {
		t.Fatalf("unexpected ALERT parse: %+v ok=%v", a, ok)
	}
	if _, ok := tz.parseMessage(msg); ok {
		t.Error("duplicate notificationId should be suppressed")
	}
}

func TestTzevaadomDrillAndUnknownThreat(t *testing.T) {
	tz := &Tzevaadom{seen: map[string]bool{}, areas: tzevaTestSet()}
	drill := []byte(`{"type":"ALERT","data":{"notificationId":"d1","time":1,"threat":5,"isDrill":true,"cities":["x"]}}`)
	if _, ok := tz.parseMessage(drill); ok {
		t.Error("drill should be dropped")
	}
	unk := []byte(`{"type":"ALERT","data":{"notificationId":"u1","time":1,"threat":99,"cities":["x"]}}`)
	if _, ok := tz.parseMessage(unk); ok {
		t.Error("unknown threat should be dropped")
	}
}

func TestTzevaadomSystemMessage(t *testing.T) {
	tz := &Tzevaadom{seen: map[string]bool{}, areas: tzevaTestSet()}
	pre := []byte(`{"type":"SYSTEM_MESSAGE","data":{"notificationId":"s1","time":"1751284800","instructionType":0,"citiesIds":[511]}}`)
	a, ok := tz.parseMessage(pre)
	if !ok || a.Category != alert.CatPreAlert || a.Kind != alert.KindPreAlert || a.Areas[0] != "אבו גוש" {
		t.Fatalf("unexpected pre-alert: %+v ok=%v", a, ok)
	}
	end := []byte(`{"type":"SYSTEM_MESSAGE","data":{"notificationId":"s2","time":"1751284800","instructionType":1,"citiesIds":[511]}}`)
	if a, ok := tz.parseMessage(end); !ok || a.Kind != alert.KindEnd {
		t.Fatalf("unexpected end: %+v ok=%v", a, ok)
	}
	unknownCity := []byte(`{"type":"SYSTEM_MESSAGE","data":{"notificationId":"s3","time":"1","instructionType":0,"citiesIds":[99999]}}`)
	if _, ok := tz.parseMessage(unknownCity); ok {
		t.Error("system message with no resolvable cities should be dropped")
	}
}
