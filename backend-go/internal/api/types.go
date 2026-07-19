package api

import (
	"time"

	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

// MatchView is the frontend-facing match payload.
type MatchView struct {
	MatchID     string     `json:"match_id"`
	FixtureID   int64      `json:"fixture_id"`
	Status      string     `json:"status"`
	IsFinal     bool       `json:"is_final"`
	FinalSource string     `json:"final_source,omitempty"`
	HomeGoals   *int32     `json:"home_goals,omitempty"`
	AwayGoals   *int32     `json:"away_goals,omitempty"`
	Seq         int32      `json:"seq"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FinalizedAt *time.Time `json:"finalized_at,omitempty"`
	StatusStale bool       `json:"status_stale"`

	StartTime          int64          `json:"start_time,omitempty"`
	CompetitionID      int32          `json:"competition_id,omitempty"`
	Competition        string         `json:"competition,omitempty"`
	FixtureGroupID     int32          `json:"fixture_group_id,omitempty"`
	Participant1ID     int32          `json:"participant1_id,omitempty"`
	Participant2ID     int32          `json:"participant2_id,omitempty"`
	Participant1IsHome bool           `json:"participant1_is_home"`
	HomeTeam           string         `json:"home_team,omitempty"`
	AwayTeam           string         `json:"away_team,omitempty"`
	SportID            int32          `json:"sport_id,omitempty"`
	CountryID          int32          `json:"country_id,omitempty"`
	Odds               *MatchOddsView `json:"odds,omitempty"`
}

// MatchOddsView is the 1X2 StablePrice line for a fixture.
type MatchOddsView struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

// SettlementStatusView reports keeper settlement progress for a matched wager.
type SettlementStatusView struct {
	State       string     `json:"state"`
	Message     string     `json:"message"`
	MatchFinal  bool       `json:"match_final"`
	SettledAt   *time.Time `json:"settled_at,omitempty"`
	TxSignature string     `json:"tx_signature,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// WagerView is the frontend-facing wager payload.
type WagerView struct {
	Pubkey       string `json:"pubkey"`
	Maker        string `json:"maker"`
	InvitedTaker string `json:"invited_taker,omitempty"`
	Taker        string `json:"taker"`
	MatchID      string `json:"match_id"`
	MakerSide    string `json:"maker_side"`
	TakerSide    string `json:"taker_side,omitempty"`
	Stake        uint64 `json:"stake"`
	Status       string `json:"status"`
}

// WagerHistoryEntryView is the frontend-facing history payload for a wallet.
type WagerHistoryEntryView struct {
	Wager            WagerView  `json:"wager"`
	Match            *MatchView `json:"match,omitempty"`
	SettlementStatus string     `json:"settlement_status"`
	Outcome          string     `json:"outcome,omitempty"`
	BackedSide       string     `json:"backed_side"`
	Opponent         string     `json:"opponent,omitempty"`
	IsMaker          bool       `json:"is_maker"`
	EventTime        int64      `json:"event_time,omitempty"`
}

type WagerHistoryPageView struct {
	Entries []WagerHistoryEntryView `json:"entries"`
	Total   int                     `json:"total"`
	Offset  int                     `json:"offset"`
	Limit   int                     `json:"limit"`
	HasMore bool                    `json:"has_more"`
}

func matchViewFromCache(m cache.Match) MatchView {
	return MatchView{
		MatchID:            m.MatchID,
		FixtureID:          m.FixtureID,
		Status:             m.GameState,
		IsFinal:            m.IsFinal,
		FinalSource:        m.FinalSource,
		HomeGoals:          m.HomeGoals,
		AwayGoals:          m.AwayGoals,
		Seq:                m.Seq,
		UpdatedAt:          m.UpdatedAt,
		FinalizedAt:        m.FinalizedAt,
		StatusStale:        cache.LiveStatusExpired(m, time.Now().UTC()),
		StartTime:          m.StartTime,
		CompetitionID:      m.CompetitionID,
		Competition:        m.Competition,
		FixtureGroupID:     m.FixtureGroupID,
		Participant1ID:     m.Participant1ID,
		Participant2ID:     m.Participant2ID,
		Participant1IsHome: m.Participant1IsHome,
		HomeTeam:           m.HomeTeam,
		AwayTeam:           m.AwayTeam,
		SportID:            m.SportID,
		CountryID:          m.CountryID,
		Odds:               oddsViewFromCache(m.Odds),
	}
}

func oddsViewFromCache(odds *cache.MatchOdds) *MatchOddsView {
	if odds == nil {
		return nil
	}
	return &MatchOddsView{
		Home: odds.Home,
		Draw: odds.Draw,
		Away: odds.Away,
	}
}

func takerSideView(w chainsol.Wager) string {
	if w.Status == chainsol.WagerStatusOpen {
		return ""
	}
	return chainsol.SideName(w.TakerSide)
}

func wagerViewFromChain(w chainsol.Wager) WagerView {
	view := WagerView{
		Pubkey:    w.Pubkey.String(),
		Maker:     w.Maker.String(),
		MatchID:   w.MatchIDString(),
		MakerSide: chainsol.SideName(w.MakerSide),
		TakerSide: takerSideView(w),
		Stake:     w.Stake,
		Status:    chainsol.StatusName(w.Status),
	}
	if !w.InvitedTaker.IsZero() && !w.InvitedTaker.Equals(chainsol.SystemProgramID) {
		view.InvitedTaker = w.InvitedTaker.String()
	}
	if w.HasCounterparty() {
		view.Taker = w.Taker.String()
	}
	return view
}
