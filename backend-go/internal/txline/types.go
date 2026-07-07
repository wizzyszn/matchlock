package txline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ScoreUpdate is a parsed TxLINE scores SSE payload (Scores schema).
// See https://txline-docs.txodds.com/api-reference/scores/get-a-real-time-server-sent-events-stream-of-scores-updates
type ScoreUpdate struct {
	// Required core fields
	FixtureID          int64  `json:"fixtureId"`
	GameState          string `json:"gameState"`
	StartTime          int64  `json:"startTime"`
	IsTeam             bool   `json:"isTeam"`
	FixtureGroupID     int32  `json:"fixtureGroupId"`
	CompetitionID      int32  `json:"competitionId"`
	CountryID          int32  `json:"countryId"`
	SportID            int32  `json:"sportId"`
	Participant1IsHome bool   `json:"participant1IsHome"`
	Participant2ID     int32  `json:"participant2Id"`
	Participant1ID     int32  `json:"participant1Id"`
	Action             string `json:"action"`
	ID                 int32  `json:"id"`
	TS                 int64  `json:"ts"`
	ConnectionID       int64  `json:"connectionId"`
	Seq                int32  `json:"seq"`

	// Optional metadata
	CoverageSecondaryData bool            `json:"coverageSecondaryData,omitempty"`
	CoverageType          string          `json:"coverageType,omitempty"`
	Confirmed             bool            `json:"confirmed,omitempty"`
	StatusID              json.RawMessage `json:"statusId,omitempty"`
	StatusBasketballID    json.RawMessage `json:"statusBasketballId,omitempty"`
	StatusSoccerID        json.RawMessage `json:"statusSoccerId,omitempty"`
	Type                  json.RawMessage `json:"type,omitempty"`
	Participant           int32           `json:"participant,omitempty"`
	Possession            int32           `json:"possession,omitempty"`
	PossessionType        json.RawMessage `json:"possessionType,omitempty"`
	Stats                 map[string]int32 `json:"stats,omitempty"`

	// Sport-specific live payloads
	Clock              *UsFootballFixtureClock `json:"clock,omitempty"`
	Down               json.RawMessage         `json:"down,omitempty"`
	InPlayInfo         json.RawMessage         `json:"inPlayInfo,omitempty"`
	KickoffInfo        json.RawMessage         `json:"kickoffInfo,omitempty"`
	Score              *UsFootballFixtureScore `json:"score,omitempty"`
	Data               json.RawMessage         `json:"data,omitempty"`
	ScoreBasketball    *BasketballFixtureScore `json:"scoreBasketball,omitempty"`
	DataBasketball     json.RawMessage         `json:"dataBasketball,omitempty"`
	ScoreSoccer        *SoccerFixtureScore     `json:"scoreSoccer,omitempty"`
	DataSoccer         *SoccerData             `json:"dataSoccer,omitempty"`
	Kickoff            json.RawMessage         `json:"kickoff,omitempty"`
	Lineups            json.RawMessage         `json:"lineups,omitempty"`
	Parti1StateSoccer  json.RawMessage         `json:"parti1StateSoccer,omitempty"`
	Parti1StateUsFootball json.RawMessage      `json:"parti1StateUsFootball,omitempty"`
	Parti1StateBasketball json.RawMessage      `json:"parti1StateBasketball,omitempty"`
	Parti2StateSoccer  json.RawMessage         `json:"parti2StateSoccer,omitempty"`
	Parti2StateUsFootball json.RawMessage      `json:"parti2StateUsFootball,omitempty"`
	Parti2StateBasketball json.RawMessage      `json:"parti2StateBasketball,omitempty"`
	PossibleEventSoccer json.RawMessage        `json:"possibleEventSoccer,omitempty"`
	PossibleEventUsFootball json.RawMessage    `json:"possibleEventUsFootball,omitempty"`

	RawEvent   string    `json:"-"`
	ReceivedAt time.Time `json:"-"`
}

// MatchID returns the canonical match identifier used across Matchlock layers.
func (s ScoreUpdate) MatchID() string {
	return strconv.FormatInt(s.FixtureID, 10)
}

// IsFinal reports whether the fixture has reached a terminal state.
func (s ScoreUpdate) IsFinal() bool {
	state := strings.ToUpper(strings.TrimSpace(s.GameState))
	switch state {
	case "F", "F1", "F2", "FET", "FPE", "FT", "FINISHED", "FULLTIME", "A", "A1", "A2":
		return true
	default:
		return false
	}
}

// HomeGoals returns home-side goals when a soccer score payload is present.
func (s ScoreUpdate) HomeGoals() (int32, bool) {
	if s.ScoreSoccer == nil {
		return 0, false
	}
	if s.Participant1IsHome {
		goals := s.ScoreSoccer.Participant1.GoalCount()
		return goals, true
	}
	goals := s.ScoreSoccer.Participant2.GoalCount()
	return goals, true
}

// AwayGoals returns away-side goals when a soccer score payload is present.
func (s ScoreUpdate) AwayGoals() (int32, bool) {
	if s.ScoreSoccer == nil {
		return 0, false
	}
	if s.Participant1IsHome {
		goals := s.ScoreSoccer.Participant2.GoalCount()
		return goals, true
	}
	goals := s.ScoreSoccer.Participant1.GoalCount()
	return goals, true
}

func (s ScoreUpdate) String() string {
	return fmt.Sprintf("fixture=%d state=%s final=%t seq=%d", s.FixtureID, s.GameState, s.IsFinal(), s.Seq)
}