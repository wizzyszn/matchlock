package txline

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// StatValidation is the TxLINE /api/scores/stat-validation response.
type StatValidation struct {
	Summary           StatValidationSummary `json:"summary"`
	SubTreeProof      []ProofNodeResponse   `json:"subTreeProof"`
	MainTreeProof     []ProofNodeResponse   `json:"mainTreeProof"`
	StatToProve       ScoreStatResponse     `json:"statToProve"`
	EventStatRoot     FlexHash              `json:"eventStatRoot"`
	StatProof         []ProofNodeResponse   `json:"statProof"`
	StatToProve2      *ScoreStatResponse    `json:"statToProve2,omitempty"`
	StatProof2        []ProofNodeResponse   `json:"statProof2,omitempty"`
}

type StatValidationSummary struct {
	FixtureID             int64                 `json:"fixtureId"`
	UpdateStats           ScoresUpdateStatsResp `json:"updateStats"`
	EventStatsSubTreeRoot FlexHash              `json:"eventStatsSubTreeRoot"`
}

// FlexHash accepts TxLINE hash fields as base64 strings or JSON byte arrays.
type FlexHash string

func (h *FlexHash) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*h = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*h = FlexHash(s)
		return nil
	}
	var arr []byte
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("flex hash: %w", err)
	}
	if len(arr) != 32 {
		return fmt.Errorf("flex hash: expected 32 bytes, got %d", len(arr))
	}
	*h = FlexHash(base64.StdEncoding.EncodeToString(arr))
	return nil
}

type ScoresUpdateStatsResp struct {
	UpdateCount   int32 `json:"updateCount"`
	MinTimestamp  int64 `json:"minTimestamp"`
	MaxTimestamp  int64 `json:"maxTimestamp"`
}

type ScoreStatResponse struct {
	Key    uint32 `json:"key"`
	Value  int32  `json:"value"`
	Period int32  `json:"period"`
}

type ProofNodeResponse struct {
	Hash           FlexHash `json:"hash"`
	IsRightSibling bool     `json:"isRightSibling"`
}