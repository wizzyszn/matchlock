package keeper

import (
	"testing"

	"github.com/matchlock/backend-go/internal/txline"
)

func TestParticipant1IsHomeFromRows(t *testing.T) {
	rows := []txline.ScoreSnapshotRow{
		{
			FixtureID:          18179763,
			GameState:          "F2",
			Seq:                941,
			Participant1IsHome: false,
			ScoreSoccer: &txline.SoccerFixtureScore{
				Participant1: txline.SoccerTotalScore{Goals: 1},
				Participant2: txline.SoccerTotalScore{Goals: 2},
			},
		},
	}
	got, ok := Participant1IsHomeFromRows(rows)
	if !ok {
		t.Fatal("expected orientation from rows")
	}
	if got {
		t.Fatalf("Participant1IsHome = true, want false")
	}
}