package cache

import "testing"

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

func TestScoreUpdateFromMatchPreservesLiveState(t *testing.T) {
	home := int32(1)
	away := int32(1)
	update, err := ScoreUpdateFromMatch(Match{
		MatchID:   "18213979",
		FixtureID: 18213979,
		GameState: "HT",
		Seq:       77,
		HomeGoals: &home,
		AwayGoals: &away,
	})
	if err != nil {
		t.Fatalf("ScoreUpdateFromMatch: %v", err)
	}
	if update.IsFinal() {
		t.Fatalf("expected live update, got state=%q", update.GameState)
	}
}
