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

// FixtureValidation is the TxLINE /api/fixtures/validation response.
type FixtureValidation struct {
	Snapshot     json.RawMessage   `json:"snapshot"`
	Summary      json.RawMessage   `json:"summary"`
	SubTreeProof []ProofNodeResponse `json:"subTreeProof"`
	MainTreeProof []ProofNodeResponse `json:"mainTreeProof"`
}

// FetchFixtureValidation loads Merkle fixture proof material from TxLINE.
func (c *Client) FetchFixtureValidation(ctx context.Context, fixtureID int64, timestamp *int64) (FixtureValidation, error) {
	u, err := url.Parse(c.baseURL + "/api/fixtures/validation")
	if err != nil {
		return FixtureValidation{}, err
	}
	q := u.Query()
	q.Set("fixtureId", strconv.FormatInt(fixtureID, 10))
	if timestamp != nil {
		q.Set("timestamp", strconv.FormatInt(*timestamp, 10))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return FixtureValidation{}, fmt.Errorf("build fixture validation request: %w", err)
	}

	resp, err := c.DoAuthenticated(ctx, req)
	if err != nil {
		return FixtureValidation{}, fmt.Errorf("fetch fixture validation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return FixtureValidation{}, fmt.Errorf("read fixture validation response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return FixtureValidation{}, fmt.Errorf("fixture validation status=%d body=%s", resp.StatusCode, truncate(body, 512))
	}

	var out FixtureValidation
	if err := json.Unmarshal(body, &out); err != nil {
		return FixtureValidation{}, fmt.Errorf("decode fixture validation: %w", err)
	}
	return out, nil
}
