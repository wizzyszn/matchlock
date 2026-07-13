package txline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// FetchScoreSnapshot returns the latest score rows for a fixture.
func (c *Client) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]ScoreSnapshotRow, error) {
	url := fmt.Sprintf("%s/api/scores/snapshot/%d", c.baseURL, fixtureID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.DoAuthenticated(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch snapshot: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapshot status=%d body=%s", resp.StatusCode, truncate(body, 512))
	}
	var rows []ScoreSnapshotRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("snapshot empty for fixture %d", fixtureID)
	}
	return rows, nil
}

// ScoreSnapshotRow is one entry from /api/scores/snapshot/{fixtureId}.
type ScoreSnapshotRow struct {
	FixtureID          int64  `json:"fixtureId"`
	FixtureIDAlt       int64  `json:"FixtureId"`
	GameState          string `json:"gameState"`
	GameStateAlt       string `json:"GameState"`
	Seq                int32  `json:"seq"`
	SeqAlt             int32  `json:"Seq"`
	Participant1IsHome bool   `json:"participant1IsHome"`
	Participant1Home   bool   `json:"Participant1IsHome"`
	Clock              *SoccerFixtureClock `json:"Clock"`
	ScoreSoccer        *SoccerFixtureScore `json:"scoreSoccer"`
	Score              *SnapshotScore `json:"Score"`
}

type SnapshotScore struct {
	Participant1 SoccerTotalScore `json:"Participant1"`
	Participant2 SoccerTotalScore `json:"Participant2"`
}

func (r ScoreSnapshotRow) Fixture() int64 {
	if r.FixtureID != 0 {
		return r.FixtureID
	}
	return r.FixtureIDAlt
}

func (r ScoreSnapshotRow) State() string {
	if r.GameState != "" {
		return r.GameState
	}
	return r.GameStateAlt
}

func (r ScoreSnapshotRow) Sequence() int32 {
	if r.Seq != 0 {
		return r.Seq
	}
	return r.SeqAlt
}

func (r ScoreSnapshotRow) HomeIsP1() bool {
	return r.Participant1IsHome || r.Participant1Home
}

// ToScoreUpdate maps the newest snapshot row into a ScoreUpdate for keeper settlement.
func (r ScoreSnapshotRow) ToScoreUpdate() (ScoreUpdate, error) {
	update := ScoreUpdate{
		FixtureID:          r.Fixture(),
		GameState:          r.State(),
		Seq:                r.Sequence(),
		Participant1IsHome: r.HomeIsP1(),
	}
	if update.FixtureID == 0 {
		return ScoreUpdate{}, fmt.Errorf("snapshot missing fixture id")
	}
	if r.ScoreSoccer != nil {
		update.ScoreSoccer = r.ScoreSoccer
		update.GameState = r.normalizedState(update.GameState)
		return update, nil
	}
	if r.Score != nil {
		update.ScoreSoccer = &SoccerFixtureScore{
			Participant1: r.Score.Participant1,
			Participant2: r.Score.Participant2,
		}
		update.GameState = r.normalizedState(update.GameState)
		return update, nil
	}
	return ScoreUpdate{}, fmt.Errorf("snapshot missing score data")
}

func (r ScoreSnapshotRow) normalizedState(state string) string {
	normalized := strings.ToLower(strings.TrimSpace(state))
	if normalized != "" && normalized != "scheduled" && normalized != "ns" && normalized != "ns2" {
		return state
	}

	score := r.score()
	if score != nil {
		if hasPenaltyPeriod(score.Participant1) || hasPenaltyPeriod(score.Participant2) {
			return "penalties"
		}
		if hasExtraTimePeriod(score.Participant1) || hasExtraTimePeriod(score.Participant2) {
			if r.Clock != nil && !r.Clock.IsRunning() && r.Clock.ElapsedSeconds() >= 6300 {
				return "htet"
			}
			return "extratime"
		}
	}

	if r.Clock != nil {
		switch {
		case r.Clock.ElapsedSeconds() >= 5400:
			return "extratime"
		case !r.Clock.IsRunning() && r.Clock.ElapsedSeconds() >= 2700:
			return "ht"
		case r.Clock.ElapsedSeconds() > 0:
			return "live"
		}
	}

	if r.Sequence() > 0 && score != nil {
		return "live"
	}

	return state
}

func (r ScoreSnapshotRow) score() *SoccerFixtureScore {
	if r.ScoreSoccer != nil {
		return r.ScoreSoccer
	}
	if r.Score != nil {
		return &SoccerFixtureScore{
			Participant1: r.Score.Participant1,
			Participant2: r.Score.Participant2,
		}
	}
	return nil
}

func hasExtraTimePeriod(score SoccerTotalScore) bool {
	return score.ET1 != nil || score.ET2 != nil || score.ETTotal != nil
}

func hasPenaltyPeriod(score SoccerTotalScore) bool {
	return score.PE != nil
}

// LatestScoreSnapshot returns the newest snapshot row with usable score data.
func LatestScoreSnapshot(rows []ScoreSnapshotRow) (ScoreSnapshotRow, error) {
	var best ScoreSnapshotRow
	var bestSeq int32
	found := false
	for i := range rows {
		update, err := rows[i].ToScoreUpdate()
		if err != nil {
			continue
		}
		seq := update.Seq
		if !found || seq >= bestSeq {
			best = rows[i]
			bestSeq = seq
			found = true
		}
	}
	if !found {
		return ScoreSnapshotRow{}, fmt.Errorf("no score snapshot row found")
	}
	return best, nil
}

// LatestFinalSnapshot picks the last snapshot row in a terminal state with scores.
func LatestFinalSnapshot(rows []ScoreSnapshotRow) (ScoreSnapshotRow, error) {
	for i := len(rows) - 1; i >= 0; i-- {
		u, err := rows[i].ToScoreUpdate()
		if err != nil {
			continue
		}
		if u.IsFinal() {
			if _, ok := u.HomeGoals(); !ok {
				continue
			}
			if _, ok := u.AwayGoals(); !ok {
				continue
			}
			return rows[i], nil
		}
	}
	return ScoreSnapshotRow{}, fmt.Errorf("no final snapshot row found")
}

// MatchIDFromFixture formats the canonical wager match_id.
func MatchIDFromFixture(fixtureID int64) string {
	return strconv.FormatInt(fixtureID, 10)
}
