package cache

import (
	"fmt"

	"github.com/matchlock/backend-go/internal/txline"
)

// ScoreUpdateFromMatch builds a TxLINE score update from cached match state.
func ScoreUpdateFromMatch(match Match) (txline.ScoreUpdate, error) {
	if match.FixtureID == 0 {
		return txline.ScoreUpdate{}, fmt.Errorf("fixture id missing for match %s", match.MatchID)
	}
	if match.HomeGoals == nil || match.AwayGoals == nil {
		return txline.ScoreUpdate{}, fmt.Errorf("scores missing for match %s", match.MatchID)
	}
	if match.Seq <= 0 {
		return txline.ScoreUpdate{}, fmt.Errorf("seq missing for match %s", match.MatchID)
	}

	state := match.GameState
	if state == "" {
		state = "FT"
	}

	p1Home := match.Participant1IsHome
	if !p1Home && match.HomeTeam == "" && match.AwayTeam == "" {
		// Legacy cache rows may lack orientation metadata.
		p1Home = true
	}

	p1Goals, p2Goals := *match.HomeGoals, *match.AwayGoals
	if !p1Home {
		p1Goals, p2Goals = *match.AwayGoals, *match.HomeGoals
	}

	return txline.ScoreUpdate{
		FixtureID:          match.FixtureID,
		GameState:          state,
		Seq:                match.Seq,
		Participant1IsHome: p1Home,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: p1Goals},
			Participant2: txline.SoccerTotalScore{Goals: p2Goals},
		},
	}, nil
}