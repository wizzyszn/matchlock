package solana

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
)

const matchStateAccountSize = 51

var matchStateDiscriminator = [8]byte{250, 209, 137, 70, 235, 96, 121, 216}

// MatchState is the durable on-chain wagering gate for a fixture.
type MatchState struct {
	Pubkey     solana.PublicKey
	MatchID    [32]byte
	MatchIDLen uint8
	IsClosed   bool
	ClosedAt   int64
	Bump       uint8
}

func (m MatchState) MatchIDString() string {
	return string(m.MatchID[:m.MatchIDLen])
}

func DecodeMatchState(pubkey solana.PublicKey, data []byte) (MatchState, error) {
	if len(data) < matchStateAccountSize {
		return MatchState{}, fmt.Errorf("match state account too small: %d", len(data))
	}
	if !bytes.Equal(data[:8], matchStateDiscriminator[:]) {
		return MatchState{}, fmt.Errorf("invalid match state discriminator")
	}

	matchIDLen := data[40]
	if matchIDLen == 0 || matchIDLen > 32 {
		return MatchState{}, fmt.Errorf("invalid match state match_id length: %d", matchIDLen)
	}
	if data[41] > 1 {
		return MatchState{}, fmt.Errorf("invalid match state closed flag: %d", data[41])
	}

	var matchID [32]byte
	copy(matchID[:], data[8:40])
	return MatchState{
		Pubkey:     pubkey,
		MatchID:    matchID,
		MatchIDLen: matchIDLen,
		IsClosed:   data[41] == 1,
		ClosedAt:   int64(binary.LittleEndian.Uint64(data[42:50])),
		Bump:       data[50],
	}, nil
}

// GetMatchState reads the durable on-chain wagering gate. The boolean is false
// when the PDA has not been initialized yet.
func (c *Client) GetMatchState(ctx context.Context, matchID string) (MatchState, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pda, _, err := FindMatchStatePDA(c.programID, []byte(matchID))
	if err != nil {
		return MatchState{}, false, fmt.Errorf("find match state PDA: %w", err)
	}
	acct, err := c.rpc.GetAccountInfo(ctx, pda)
	if err != nil {
		return MatchState{}, false, fmt.Errorf("get match state %s: %w", matchID, err)
	}
	if acct == nil || acct.Value == nil {
		return MatchState{}, false, nil
	}
	if !acct.Value.Owner.Equals(c.programID) {
		return MatchState{}, false, fmt.Errorf(
			"match state %s has unexpected owner %s",
			matchID,
			acct.Value.Owner,
		)
	}

	state, err := DecodeMatchState(pda, acct.Value.Data.GetBinary())
	if err != nil {
		return MatchState{}, false, fmt.Errorf("decode match state %s: %w", matchID, err)
	}
	if state.MatchIDString() != matchID {
		return MatchState{}, false, fmt.Errorf(
			"match state id mismatch: got %q want %q",
			state.MatchIDString(),
			matchID,
		)
	}
	return state, true, nil
}
