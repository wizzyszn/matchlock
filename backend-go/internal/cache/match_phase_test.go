package cache

import (
	"testing"
	"time"
)

func TestInferFinalState(t *testing.T) {
	now := time.UnixMilli(1782950400000 + int64((2 * time.Hour).Milliseconds()))

	match := Match{
		MatchID:   "18172379",
		GameState: "scheduled",
		StartTime: 1782950400000,
		Seq:       1057,
	}

	if !FinalVerificationEligible(match, now) {
		t.Fatal("expected verification eligible")
	}
}

func TestInferFinalStateSkipsRecentKickoff(t *testing.T) {
	now := time.UnixMilli(1782950400000 + int64((30 * time.Minute).Milliseconds()))

	match := Match{
		StartTime: 1782950400000,
	}
	if FinalVerificationEligible(match, now) {
		t.Fatal("expected not verification eligible within match window")
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

	if !FinalVerificationEligible(match, now) {
		t.Fatal("expected verification eligible even without goals")
	}
}
