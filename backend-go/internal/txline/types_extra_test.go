package txline

import "testing"

func TestScoreUpdateHelpers(t *testing.T) {
	update := ScoreUpdate{
		FixtureID:          42,
		GameState:          "HT",
		Participant1IsHome: true,
		ScoreSoccer: &SoccerFixtureScore{
			Participant1: SoccerTotalScore{Goals: 1},
			Participant2: SoccerTotalScore{Goals: 2},
		},
	}
	if update.MatchID() != "42" {
		t.Fatalf("match_id = %q", update.MatchID())
	}
	if update.IsFinal() {
		t.Fatal("HT should not be final")
	}
	if home, ok := update.HomeGoals(); !ok || home != 1 {
		t.Fatalf("home = %d ok=%v", home, ok)
	}
	if away, ok := update.AwayGoals(); !ok || away != 2 {
		t.Fatalf("away = %d ok=%v", away, ok)
	}
	if update.String() == "" {
		t.Fatal("expected string")
	}
}