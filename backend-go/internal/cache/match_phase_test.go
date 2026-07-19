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

func TestLiveStatusExpired(t *testing.T) {
	kickoff := int64(1782950400000)
	match := Match{StartTime: kickoff}

	if LiveStatusExpired(match, time.UnixMilli(kickoff).Add(3*time.Hour)) {
		t.Fatal("three-hour fixture should not be marked stale")
	}
	if !LiveStatusExpired(match, time.UnixMilli(kickoff).Add(4*time.Hour)) {
		t.Fatal("four-hour fixture should be marked stale")
	}

	match.IsFinal = true
	if LiveStatusExpired(match, time.UnixMilli(kickoff).Add(24*time.Hour)) {
		t.Fatal("final fixture should never be marked stale")
	}
}
