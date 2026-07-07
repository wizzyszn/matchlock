package solana

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matchlock/backend-go/internal/txline"
)

func TestValidationFromAPIFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "txline_proof_response.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var validation txline.StatValidation
	if err := json.Unmarshal(raw, &validation); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	args, _, err := ValidationFromAPI(validation)
	if err != nil {
		t.Fatalf("ValidationFromAPI: %v", err)
	}
	if args.FixtureSummary.FixtureID != 17952170 {
		t.Fatalf("fixture id = %d", args.FixtureSummary.FixtureID)
	}
	if args.TS != 1700000000000 {
		t.Fatalf("ts = %d", args.TS)
	}
	if args.StatA.StatToProve.Key != 1002 {
		t.Fatalf("stat key = %d", args.StatA.StatToProve.Key)
	}
}