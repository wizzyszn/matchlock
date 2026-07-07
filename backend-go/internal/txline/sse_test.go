package txline

import (
	"strings"
	"testing"
)

func TestParseSSEBlock(t *testing.T) {
	block := "event: score\ndata: {\"fixtureId\":123}\n"
	msg, ok := ParseSSEBlock(block)
	if !ok {
		t.Fatal("expected parsed message")
	}
	if msg.Event != "score" {
		t.Fatalf("event = %q", msg.Event)
	}
	if msg.Data != `{"fixtureId":123}` {
		t.Fatalf("data = %q", msg.Data)
	}
}

func TestParseScoreUpdateSkipsHeartbeat(t *testing.T) {
	raw := `{"Ts":12345}`
	update, err := parseScoreUpdate(SSEMessage{Event: "heartbeat", Data: raw})
	if err == nil {
		t.Fatalf("expected heartbeat without fixtureId to fail, got %#v", update)
	}
}

func TestParseScoreUpdate(t *testing.T) {
	raw := `{
		"fixtureId": 17952170,
		"gameState": "F2",
		"action": "score",
		"seq": 10,
		"ts": 1700000000000,
		"participant1IsHome": true,
		"scoreSoccer": {
			"Participant1": {"Goals": 2},
			"Participant2": {"Goals": 1}
		}
	}`
	update, err := parseScoreUpdate(SSEMessage{Data: raw, Event: "score"})
	if err != nil {
		t.Fatalf("parseScoreUpdate: %v", err)
	}
	if update.MatchID() != "17952170" {
		t.Fatalf("match_id = %q", update.MatchID())
	}
	if !update.IsFinal() {
		t.Fatal("expected final state")
	}
	home, ok := update.HomeGoals()
	if !ok || home != 2 {
		t.Fatalf("home goals = %d ok=%v", home, ok)
	}
	away, ok := update.AwayGoals()
	if !ok || away != 1 {
		t.Fatalf("away goals = %d ok=%v", away, ok)
	}
}

func TestReadSSEMultipleBlocks(t *testing.T) {
	input := strings.NewReader("data: one\n\ndata: two\n\n")
	out := make(chan SSEMessage, 4)
	if err := ReadSSE(input, out); err != nil {
		t.Fatalf("ReadSSE: %v", err)
	}
	close(out)

	var data []string
	for msg := range out {
		data = append(data, msg.Data)
	}
	if len(data) != 2 || data[0] != "one" || data[1] != "two" {
		t.Fatalf("messages = %#v", data)
	}
}