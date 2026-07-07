package txline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseScoreUpdateFixtures(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantFinal bool
		wantErr   bool
		wantMatch string
	}{
		{
			name:      "final",
			fixture:   "sse_match_final.json",
			wantFinal: true,
			wantMatch: "17952170",
		},
		{
			name:      "live",
			fixture:   "sse_match_live.json",
			wantFinal: false,
			wantMatch: "17952170",
		},
		{
			name:    "malformed",
			fixture: "sse_malformed.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, err := parseScoreUpdate(SSEMessage{
				Event: "score",
				Data:  string(loadFixture(t, tt.fixture)),
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseScoreUpdate: %v", err)
			}
			if update.MatchID() != tt.wantMatch {
				t.Fatalf("match_id = %q, want %q", update.MatchID(), tt.wantMatch)
			}
			if update.IsFinal() != tt.wantFinal {
				t.Fatalf("is_final = %v, want %v", update.IsFinal(), tt.wantFinal)
			}
		})
	}
}

func TestParseScoreUpdateValidation(t *testing.T) {
	tests := []struct {
		name    string
		msg     SSEMessage
		wantErr string
	}{
		{
			name:    "empty data",
			msg:     SSEMessage{Event: "score"},
			wantErr: "empty data field",
		},
		{
			name:    "missing fixture id",
			msg:     SSEMessage{Event: "score", Data: `{"gameState":"HT"}`},
			wantErr: "missing fixtureId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseScoreUpdate(tt.msg)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestIsFinalStates(t *testing.T) {
	finalStates := []string{"F", "F2", "FET", "FPE", "FT", "FINISHED", "FULLTIME", "A", "A2"}
	for _, state := range finalStates {
		update := ScoreUpdate{GameState: state}
		if !update.IsFinal() {
			t.Fatalf("state %q should be final", state)
		}
	}
	live := ScoreUpdate{GameState: "HT"}
	if live.IsFinal() {
		t.Fatal("HT should not be final")
	}
}

func TestHomeAwayGoalsParticipantOrder(t *testing.T) {
	homeFirst := ScoreUpdate{
		Participant1IsHome: true,
		ScoreSoccer: &SoccerFixtureScore{
			Participant1: SoccerTotalScore{Goals: 3},
			Participant2: SoccerTotalScore{Goals: 2},
		},
	}
	home, ok := homeFirst.HomeGoals()
	if !ok || home != 3 {
		t.Fatalf("home = %d ok=%v", home, ok)
	}
	away, ok := homeFirst.AwayGoals()
	if !ok || away != 2 {
		t.Fatalf("away = %d ok=%v", away, ok)
	}

	awayFirst := ScoreUpdate{
		Participant1IsHome: false,
		ScoreSoccer: &SoccerFixtureScore{
			Participant1: SoccerTotalScore{Goals: 1},
			Participant2: SoccerTotalScore{Goals: 4},
		},
	}
	home, ok = awayFirst.HomeGoals()
	if !ok || home != 4 {
		t.Fatalf("home = %d ok=%v", home, ok)
	}
	away, ok = awayFirst.AwayGoals()
	if !ok || away != 1 {
		t.Fatalf("away = %d ok=%v", away, ok)
	}
}