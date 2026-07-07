package txline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	oddsMarket1X2       = "1X2_PARTICIPANT_RESULT"
	oddsPriceScale      = 1000.0
	oddsBookmakerStable = "TXLineStablePriceDemargined"
)

// OddsPayload is one market line from /api/odds/snapshot/{fixtureId}.
type OddsPayload struct {
	FixtureID        int64    `json:"FixtureId"`
	MessageID        string   `json:"MessageId"`
	Ts               int64    `json:"Ts"`
	Bookmaker        string   `json:"Bookmaker"`
	BookmakerID      int32    `json:"BookmakerId"`
	SuperOddsType    string   `json:"SuperOddsType"`
	GameState        string   `json:"GameState"`
	InRunning        bool     `json:"InRunning"`
	MarketParameters string   `json:"MarketParameters"`
	MarketPeriod     string   `json:"MarketPeriod"`
	PriceNames       []string `json:"PriceNames"`
	Prices           []int32  `json:"Prices"`
}

// MatchOdds is the demargined 1X2 StablePrice line exposed to the API.
type MatchOdds struct {
	Home float64 `json:"home"`
	Draw float64 `json:"draw"`
	Away float64 `json:"away"`
}

// FetchOddsSnapshot returns latest odds market lines for a fixture.
func (c *Client) FetchOddsSnapshot(ctx context.Context, fixtureID int64) ([]OddsPayload, error) {
	return c.fetchOdds(ctx, fmt.Sprintf("%s/api/odds/snapshot/%d", c.baseURL, fixtureID))
}

// FetchOddsSnapshotAsOf returns historical odds at a specific Unix timestamp (ms).
func (c *Client) FetchOddsSnapshotAsOf(ctx context.Context, fixtureID int64, asOf int64) ([]OddsPayload, error) {
	url := fmt.Sprintf("%s/api/odds/snapshot/%d?asOf=%d", c.baseURL, fixtureID, asOf)
	return c.fetchOdds(ctx, url)
}

// FetchOddsUpdates returns live odds from the in-memory 5-minute cache for a fixture.
func (c *Client) FetchOddsUpdates(ctx context.Context, fixtureID int64) ([]OddsPayload, error) {
	return c.fetchOdds(ctx, fmt.Sprintf("%s/api/odds/updates/%d", c.baseURL, fixtureID))
}

func (c *Client) fetchOdds(ctx context.Context, url string) ([]OddsPayload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.DoAuthenticated(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch odds: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("odds status=%d body=%s", resp.StatusCode, truncate(body, 512))
	}

	var rows []OddsPayload
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode odds: %w", err)
	}
	return rows, nil
}

// Parse1X2Odds extracts the full-time 1X2 StablePrice line from snapshot rows.
func Parse1X2Odds(rows []OddsPayload) (MatchOdds, bool) {
	var best *OddsPayload
	for i := range rows {
		row := &rows[i]
		if row.SuperOddsType != oddsMarket1X2 {
			continue
		}
		if row.MarketParameters != "" || row.MarketPeriod != "" {
			continue
		}
		if best == nil || oddsRowRank(*row) > oddsRowRank(*best) {
			best = row
		}
	}
	if best == nil {
		return MatchOdds{}, false
	}

	home, draw, away, ok := priceTriple(best.PriceNames, best.Prices)
	if !ok {
		return MatchOdds{}, false
	}
	return MatchOdds{Home: home, Draw: draw, Away: away}, true
}

func priceTriple(names []string, prices []int32) (home, draw, away float64, ok bool) {
	if len(names) != len(prices) || len(prices) < 3 {
		return 0, 0, 0, false
	}
	index := map[string]int{}
	for i, name := range names {
		index[strings.ToLower(strings.TrimSpace(name))] = i
	}
	homeIdx, okHome := index["part1"]
	drawIdx, okDraw := index["draw"]
	awayIdx, okAway := index["part2"]
	if !okHome || !okDraw || !okAway {
		return 0, 0, 0, false
	}
	return scaleOddsPrice(prices[homeIdx]),
		scaleOddsPrice(prices[drawIdx]),
		scaleOddsPrice(prices[awayIdx]),
		true
}

func oddsRowRank(row OddsPayload) int64 {
	rank := row.Ts
	if !row.InRunning {
		rank += 1_000_000_000_000
	}
	return rank
}

func scaleOddsPrice(raw int32) float64 {
	if raw <= 0 {
		return 0
	}
	return float64(raw) / oddsPriceScale
}