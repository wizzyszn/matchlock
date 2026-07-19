package cache

import (
	"errors"
	"time"

	"github.com/matchlock/backend-go/internal/txline"
)

// ErrMatchNotFound is returned when a match id is absent from the store.
var ErrMatchNotFound = errors.New("match not found")

// ErrSettlementNotFound is returned when no settlement record exists for a wager.
var ErrSettlementNotFound = errors.New("settlement not found")

// ErrPendingSettlementNotFound is returned when a wager is not in the retry queue.
var ErrPendingSettlementNotFound = errors.New("pending settlement not found")

// Match is the cached view of a TxLINE fixture used by the API and keeper.
type Match struct {
	MatchID     string     `json:"match_id"`
	FixtureID   int64      `json:"fixture_id"`
	GameState   string     `json:"game_state"`
	IsFinal     bool       `json:"is_final"`
	FinalSource string     `json:"final_source,omitempty"`
	HomeGoals   *int32     `json:"home_goals,omitempty"`
	AwayGoals   *int32     `json:"away_goals,omitempty"`
	Seq         int32      `json:"seq"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty"`

	// Schedule metadata (from fixtures snapshot and/or scores stream)
	StartTime          int64      `json:"start_time,omitempty"`
	CompetitionID      int32      `json:"competition_id,omitempty"`
	Competition        string     `json:"competition,omitempty"`
	FixtureGroupID     int32      `json:"fixture_group_id,omitempty"`
	Participant1ID     int32      `json:"participant1_id,omitempty"`
	Participant2ID     int32      `json:"participant2_id,omitempty"`
	Participant1IsHome bool       `json:"participant1_is_home,omitempty"`
	HomeTeam           string     `json:"home_team,omitempty"`
	AwayTeam           string     `json:"away_team,omitempty"`
	SportID            int32      `json:"sport_id,omitempty"`
	CountryID          int32      `json:"country_id,omitempty"`
	Odds               *MatchOdds `json:"odds,omitempty"`
}

// MatchOdds is the cached 1X2 StablePrice line for a fixture.
type MatchOdds struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

// SettlementRecord tracks a successful on-chain settlement for idempotency.
type SettlementRecord struct {
	MatchID     string    `json:"match_id"`
	WagerPubkey string    `json:"wager_pubkey"`
	TxSignature string    `json:"tx_signature"`
	SettledAt   time.Time `json:"settled_at"`
}

// ApplyScoreUpdate merges a live scores SSE event onto cached match state.
// Once a match is txline-verified final, non-final updates are discarded to
// prevent stale snapshot data from regressing the definitive result.
func ApplyScoreUpdate(existing Match, update txline.ScoreUpdate) Match {
	if existing.IsFinal && existing.FinalSource == FinalSourceTxline && !update.IsFinal() {
		return existing
	}

	out := existing
	out.MatchID = update.MatchID()
	out.FixtureID = update.FixtureID
	out.GameState = update.GameState
	if update.IsFinal() {
		out.IsFinal = true
		out.FinalSource = FinalSourceTxline
	}
	if update.Seq > 0 {
		out.Seq = update.Seq
	}
	out.UpdatedAt = time.Now().UTC()

	if update.StartTime > 0 {
		out.StartTime = update.StartTime
	}
	if update.CompetitionID > 0 {
		out.CompetitionID = update.CompetitionID
	}
	if update.FixtureGroupID > 0 {
		out.FixtureGroupID = update.FixtureGroupID
	}
	if update.Participant1ID > 0 {
		out.Participant1ID = update.Participant1ID
	}
	if update.Participant2ID > 0 {
		out.Participant2ID = update.Participant2ID
	}
	out.Participant1IsHome = update.Participant1IsHome
	if update.SportID > 0 {
		out.SportID = update.SportID
	}
	if update.CountryID > 0 {
		out.CountryID = update.CountryID
	}

	if home, ok := update.HomeGoals(); ok {
		v := home
		out.HomeGoals = &v
	}
	if away, ok := update.AwayGoals(); ok {
		v := away
		out.AwayGoals = &v
	}
	if update.IsFinal() {
		now := time.Now().UTC()
		out.FinalizedAt = &now
	}
	return out
}

// ApplyFixtureSchedule merges schedule metadata without clobbering live score state.
func ApplyFixtureSchedule(existing Match, fixture txline.Fixture) Match {
	out := existing
	out.MatchID = fixture.MatchID()
	out.FixtureID = fixture.FixtureID
	out.StartTime = fixture.StartTime
	out.CompetitionID = fixture.CompetitionID
	out.Competition = fixture.Competition
	out.FixtureGroupID = fixture.FixtureGroupID
	out.Participant1ID = fixture.Participant1ID
	out.Participant2ID = fixture.Participant2ID
	out.HomeTeam = fixture.HomeTeam()
	out.AwayTeam = fixture.AwayTeam()
	out.Participant1IsHome = fixture.Participant1IsHome

	if out.Seq == 0 && out.GameState == "" {
		out.GameState = "scheduled"
		out.IsFinal = false
		out.UpdatedAt = time.Now().UTC()
	}
	return out
}

// ApplyMatchOdds merges 1X2 odds onto a cached match.
func ApplyMatchOdds(existing Match, odds MatchOdds) Match {
	out := existing
	out.Odds = &odds
	return out
}
