package txline

import (
	"encoding/json"
	"testing"
)

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

func TestSnapshotToScoreUpdateRecognizesFinalizedAction(t *testing.T) {
	row := ScoreSnapshotRow{
		FixtureIDAlt:     18237038,
		GameStateAlt:     "scheduled",
		StartTimeAlt:     1784055600000,
		ActionAlt:        "game_finalised",
		StatusIDAlt:      json.RawMessage("100"),
		TSAlt:            1784063054751,
		SeqAlt:           1026,
		Participant1Home: true,
		Score: &SnapshotScore{
			Participant1: SoccerTotalScore{Total: &SoccerScore{Goals: 0}},
			Participant2: SoccerTotalScore{Total: &SoccerScore{Goals: 2}},
		},
	}

	update, err := row.ToScoreUpdate()
	if err != nil {
		t.Fatalf("ToScoreUpdate: %v", err)
	}
	if !update.IsFinal() || update.GameState != "FT" {
		t.Fatalf("final update = %#v", update)
	}
	if update.StartTime != row.StartTimeAlt || update.TS != row.TSAlt {
		t.Fatalf("timestamps not preserved: %#v", update)
	}
	home, okHome := update.HomeGoals()
	away, okAway := update.AwayGoals()
	if !okHome || !okAway || home != 0 || away != 2 {
		t.Fatalf("score = %d-%d ok=%v/%v", home, away, okHome, okAway)
	}
}

func TestScoreSnapshotFinalMarkersDecodeFromAPIShape(t *testing.T) {
	raw := []byte(`[{
		"FixtureId": 18237038,
		"GameState": "scheduled",
		"StartTime": 1784055600000,
		"Action": "game_finalised",
		"StatusId": 100,
		"Ts": 1784063054751,
		"Seq": 1026,
		"Participant1IsHome": true,
		"Score": {
			"Participant1": {"Total": {"Goals": 0}},
			"Participant2": {"Total": {"Goals": 2}}
		}
	}]`)

	var rows []ScoreSnapshotRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	row, err := LatestFinalSnapshot(rows)
	if err != nil {
		t.Fatalf("LatestFinalSnapshot: %v", err)
	}
	update, err := row.ToScoreUpdate()
	if err != nil {
		t.Fatalf("ToScoreUpdate: %v", err)
	}
	if !update.IsFinal() || update.Seq != 1026 {
		t.Fatalf("update = %#v", update)
	}
}

func TestSnapshotToScoreUpdateRecognizesStringFinalStatus(t *testing.T) {
	row := ScoreSnapshotRow{
		FixtureID: 42,
		GameState: "scheduled",
		StatusID:  json.RawMessage(`"100"`),
		Seq:       8,
		ScoreSoccer: &SoccerFixtureScore{
			Participant1: SoccerTotalScore{Goals: 1},
			Participant2: SoccerTotalScore{Goals: 0},
		},
	}

	update, err := row.ToScoreUpdate()
	if err != nil {
		t.Fatalf("ToScoreUpdate: %v", err)
	}
	if !update.IsFinal() {
		t.Fatalf("update should be final: %#v", update)
	}
}

func TestLatestFinalSnapshotUsesHighestSequence(t *testing.T) {
	rows := []ScoreSnapshotRow{
		{
			FixtureID: 42,
			GameState: "scheduled",
			Action:    "game_finalised",
			Seq:       12,
			ScoreSoccer: &SoccerFixtureScore{
				Participant1: SoccerTotalScore{Goals: 2},
				Participant2: SoccerTotalScore{Goals: 0},
			},
		},
		{
			FixtureID: 42,
			GameState: "FT",
			Seq:       9,
			ScoreSoccer: &SoccerFixtureScore{
				Participant1: SoccerTotalScore{Goals: 1},
				Participant2: SoccerTotalScore{Goals: 0},
			},
		},
	}

	got, err := LatestFinalSnapshot(rows)
	if err != nil {
		t.Fatalf("LatestFinalSnapshot: %v", err)
	}
	if got.Sequence() != 12 {
		t.Fatalf("seq = %d, want 12", got.Sequence())
	}
}
