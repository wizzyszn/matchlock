package cache

import (
	"testing"
	"time"
)

func TestInferFinalState(t *testing.T) {
	home := int32(2)
	away := int32(0)
	now := time.UnixMilli(1782950400000 + int64((2 * time.Hour).Milliseconds()))

	match := Match{
		MatchID:   "18172379",
		GameState: "scheduled",
		StartTime: 1782950400000,
		HomeGoals: &home,
		AwayGoals: &away,
		Seq:       1057,
	}

	got := InferFinalState(match, now)
	if !got.IsFinal {
		t.Fatal("expected final")
	}
	if got.GameState != "FT" {
		t.Fatalf("game_state = %q", got.GameState)
	}
}

func TestInferFinalStateSkipsRecentKickoff(t *testing.T) {
	home := int32(1)
	away := int32(0)
	now := time.UnixMilli(1782950400000 + int64((30 * time.Minute).Milliseconds()))

	match := Match{
		StartTime: 1782950400000,
		HomeGoals: &home,
		AwayGoals: &away,
	}
	got := InferFinalState(match, now)
	if got.IsFinal {
		t.Fatal("expected not final within match window")
	}
}

func TestInferFinalStateWithoutGoals(t *testing.T) {
	now := time.UnixMilli(1782950400000 + int64((2 * time.Hour).Milliseconds()))
	match := Match{
		MatchID:   "18172380",
		GameState: "scheduled",
		StartTime: 1782950400000,
		Seq:       1057,
	}

	got := InferFinalState(match, now)
	if !got.IsFinal {
		t.Fatal("expected final even without goals")
	}
	if got.GameState != "FT" {
		t.Fatalf("game_state = %q", got.GameState)
	}
}		