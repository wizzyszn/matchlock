package txline

import "testing"

func TestScoreUpdateStringAndGoalsMissing(t *testing.T) {
	update := ScoreUpdate{FixtureID: 42, GameState: "HT"}
	if update.String() == "" {
		t.Fatal("expected string output")
	}
	if _, ok := update.HomeGoals(); ok {
		t.Fatal("expected missing home goals")
	}
	if _, ok := update.AwayGoals(); ok {
		t.Fatal("expected missing away goals")
	}
}