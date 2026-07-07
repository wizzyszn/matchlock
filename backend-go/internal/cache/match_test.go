package cache

import (
	"testing"

	"github.com/matchlock/backend-go/internal/txline"
)

func TestApplyScoreUpdatePreservesScheduleMetadata(t *testing.T) {
	existing := Match{
		MatchID:   "18172379",
		FixtureID: 18172379,
		HomeTeam:  "USA",
		AwayTeam:  "Bosnia & Herzegovina",
		Competition: "World Cup",
		StartTime: 1783036800000,
	}
	update := txline.ScoreUpdate{
		FixtureID:          18172379,
		GameState:          "HT",
		Seq:                12,
		Participant1IsHome: true,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 1},
			Participant2: txline.SoccerTotalScore{Goals: 0},
		},
	}

	got := ApplyScoreUpdate(existing, update)
	if got.HomeTeam != "USA" || got.AwayTeam != "Bosnia & Herzegovina" {
		t.Fatalf("schedule names lost: %#v", got)
	}
	if got.Seq != 12 || got.GameState != "HT" {
		t.Fatalf("live state = %#v", got)
	}
	if got.HomeGoals == nil || *got.HomeGoals != 1 {
		t.Fatalf("home_goals = %#v", got.HomeGoals)
	}
}

func TestApplyFixtureScheduleDoesNotClobberLiveState(t *testing.T) {
	existing := Match{
		MatchID:   "18172379",
		FixtureID: 18172379,
		GameState: "HT",
		Seq:       12,
	}
	fixture := txline.Fixture{
		FixtureID:          18172379,
		StartTime:          1783036800000,
		Competition:        "World Cup",
		Participant1:       "USA",
		Participant2:       "Bosnia & Herzegovina",
		Participant1IsHome: true,
	}

	got := ApplyFixtureSchedule(existing, fixture)
	if got.GameState != "HT" || got.Seq != 12 {
		t.Fatalf("live state clobbered: %#v", got)
	}
	if got.HomeTeam != "USA" || got.Competition != "World Cup" {
		t.Fatalf("schedule metadata = %#v", got)
	}
}

func TestApplyFixtureScheduleSeedsScheduledMatch(t *testing.T) {
	fixture := txline.Fixture{
		FixtureID:          18172379,
		StartTime:          1783036800000,
		Competition:        "World Cup",
		Participant1:       "USA",
		Participant2:       "Bosnia & Herzegovina",
		Participant1IsHome: true,
	}
	got := ApplyFixtureSchedule(Match{}, fixture)
	if got.GameState != "scheduled" {
		t.Fatalf("game_state = %q", got.GameState)
	}
	if got.HomeTeam != "USA" {
		t.Fatalf("home_team = %q", got.HomeTeam)
	}
}