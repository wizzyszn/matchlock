package cache

import (
	"testing"
	"time"
)

func TestScoreUpdateFromMatch(t *testing.T) {
	home := int32(2)
	away := int32(1)
	update, err := ScoreUpdateFromMatch(Match{
		MatchID:   "18179763",
		FixtureID: 18179763,
		GameState: "F2",
		Seq:       941,
		HomeGoals: &home,
		AwayGoals: &away,
	})
	if err != nil {
		t.Fatalf("ScoreUpdateFromMatch: %v", err)
	}
	if !update.IsFinal() {
		t.Fatal("expected final update")
	}
	h, ok := update.HomeGoals()
	if !ok || h != 2 {
		t.Fatalf("home goals = %d ok=%v", h, ok)
	}
}

func TestInferFinalStateSetsSource(t *testing.T) {
	home := int32(1)
	away := int32(0)
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	match := InferFinalState(Match{
		MatchID:   "1",
		StartTime: now.Add(-2 * time.Hour).UnixMilli(),
		HomeGoals: &home,
		AwayGoals: &away,
	}, now)
	if !match.IsFinal || match.FinalSource != FinalSourceInferred {
		t.Fatalf("match = %#v", match)
	}
}