package txline

import "testing"

func TestSnapshotToScoreUpdateNormalizesExtraTimeState(t *testing.T) {
	row := ScoreSnapshotRow{
		FixtureIDAlt:     18213979,
		GameStateAlt:     "scheduled",
		SeqAlt:           1007,
		Participant1Home: true,
		Clock:            &SoccerFixtureClock{RunningAlt: true, SecondsAlt: 6471},
		Score: &SnapshotScore{
			Participant1: SoccerTotalScore{
				H1:      &SoccerScore{Goals: 1},
				HT:      &SoccerScore{Goals: 1},
				ET1:     &SoccerScore{Corners: 1},
				ETTotal: &SoccerScore{Corners: 1},
				Total:   &SoccerScore{Goals: 1, Corners: 6},
			},
			Participant2: SoccerTotalScore{
				H1:      &SoccerScore{Goals: 1, Corners: 2},
				HT:      &SoccerScore{Goals: 1, Corners: 2},
				ET1:     &SoccerScore{Goals: 1, Corners: 1},
				ETTotal: &SoccerScore{Goals: 1, Corners: 1},
				Total:   &SoccerScore{Goals: 2, Corners: 4},
			},
		},
	}

	update, err := row.ToScoreUpdate()
	if err != nil {
		t.Fatalf("ToScoreUpdate: %v", err)
	}
	if update.GameState != "extratime" {
		t.Fatalf("game_state = %q, want extratime", update.GameState)
	}
	away, ok := update.AwayGoals()
	if !ok || away != 2 {
		t.Fatalf("away = %d ok=%v", away, ok)
	}
}

func TestSnapshotToScoreUpdateNormalizesScheduledLiveState(t *testing.T) {
	row := ScoreSnapshotRow{
		FixtureIDAlt:     42,
		GameStateAlt:     "scheduled",
		SeqAlt:           18,
		Participant1Home: true,
		Clock:            &SoccerFixtureClock{RunningAlt: true, SecondsAlt: 1810},
		Score: &SnapshotScore{
			Participant1: SoccerTotalScore{Total: &SoccerScore{Goals: 1}},
			Participant2: SoccerTotalScore{Total: &SoccerScore{Goals: 0}},
		},
	}

	update, err := row.ToScoreUpdate()
	if err != nil {
		t.Fatalf("ToScoreUpdate: %v", err)
	}
	if update.GameState != "live" {
		t.Fatalf("game_state = %q, want live", update.GameState)
	}
}
