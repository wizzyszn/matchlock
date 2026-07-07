package txline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// FetchStatValidation loads Merkle proof material for on-chain settlement.
func (c *Client) FetchStatValidation(ctx context.Context, fixtureID int64, seq int32, statKey uint32) (StatValidation, error) {
	u, err := url.Parse(c.baseURL + "/api/scores/stat-validation")
	if err != nil {
		return StatValidation{}, err
	}
	q := u.Query()
	q.Set("fixtureId", strconv.FormatInt(fixtureID, 10))
	q.Set("seq", strconv.FormatInt(int64(seq), 10))
	q.Set("statKey", strconv.FormatUint(uint64(statKey), 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return StatValidation{}, fmt.Errorf("build proof request: %w", err)
	}

	resp, err := c.DoAuthenticated(ctx, req)
	if err != nil {
		return StatValidation{}, fmt.Errorf("fetch stat validation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return StatValidation{}, fmt.Errorf("read proof response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return StatValidation{}, fmt.Errorf("stat validation status=%d body=%s", resp.StatusCode, truncate(body, 512))
	}

	var out StatValidation
	if err := json.Unmarshal(body, &out); err != nil {
		return StatValidation{}, fmt.Errorf("decode stat validation: %w", err)
	}
	if out.Summary.FixtureID == 0 {
		return StatValidation{}, fmt.Errorf("stat validation missing fixture summary")
	}
	return out, nil
}