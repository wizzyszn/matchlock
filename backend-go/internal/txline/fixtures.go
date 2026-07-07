package txline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// Fixture is one row from GET /api/fixtures/snapshot.
type Fixture struct {
	Ts                 int64  `json:"Ts"`
	StartTime          int64  `json:"StartTime"`
	Competition        string `json:"Competition"`
	CompetitionID      int32  `json:"CompetitionId"`
	FixtureGroupID     int32  `json:"FixtureGroupId"`
	Participant1ID     int32  `json:"Participant1Id"`
	Participant1       string `json:"Participant1"`
	Participant2ID     int32  `json:"Participant2Id"`
	Participant2       string `json:"Participant2"`
	FixtureID          int64  `json:"FixtureId"`
	Participant1IsHome bool   `json:"Participant1IsHome"`
}

// MatchID returns the canonical wager match_id for this fixture.
func (f Fixture) MatchID() string {
	return strconv.FormatInt(f.FixtureID, 10)
}

// HomeTeam returns the home participant name.
func (f Fixture) HomeTeam() string {
	if f.Participant1IsHome {
		return f.Participant1
	}
	return f.Participant2
}

// AwayTeam returns the away participant name.
func (f Fixture) AwayTeam() string {
	if f.Participant1IsHome {
		return f.Participant2
	}
	return f.Participant1
}

// FetchFixturesSnapshot returns upcoming fixtures from TxLINE schedule coverage.
// startEpochDay is optional; when nil the API defaults to the current UTC day.
func (c *Client) FetchFixturesSnapshot(ctx context.Context, startEpochDay *int) ([]Fixture, error) {
	url := c.baseURL + "/api/fixtures/snapshot"
	if startEpochDay != nil {
		url = fmt.Sprintf("%s?startEpochDay=%d", url, *startEpochDay)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.DoAuthenticated(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch fixtures snapshot: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fixtures snapshot status=%d body=%s", resp.StatusCode, truncate(body, 512))
	}

	var fixtures []Fixture
	if err := json.Unmarshal(body, &fixtures); err != nil {
		return nil, fmt.Errorf("decode fixtures snapshot: %w", err)
	}
	return fixtures, nil
}