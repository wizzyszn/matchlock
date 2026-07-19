package solana

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
)

func buildMatchStateAccountData(matchID string, closed bool, closedAt int64) []byte {
	data := make([]byte, matchStateAccountSize)
	copy(data[:8], matchStateDiscriminator[:])
	copy(data[8:40], []byte(matchID))
	data[40] = uint8(len(matchID))
	if closed {
		data[41] = 1
	}
	binary.LittleEndian.PutUint64(data[42:50], uint64(closedAt))
	data[50] = 254
	return data
}

func TestDecodeMatchState(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	state, err := DecodeMatchState(pubkey, buildMatchStateAccountData("18237038", true, 1784063054))
	if err != nil {
		t.Fatalf("DecodeMatchState: %v", err)
	}
	if state.MatchIDString() != "18237038" || !state.IsClosed || state.ClosedAt != 1784063054 {
		t.Fatalf("state = %#v", state)
	}
}

func TestDecodeMatchStateRejectsInvalidData(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	if _, err := DecodeMatchState(pubkey, []byte{1, 2, 3}); err == nil {
		t.Fatal("expected size error")
	}

	data := buildMatchStateAccountData("18237038", false, 0)
	data[0] ^= 0xff
	if _, err := DecodeMatchState(pubkey, data); err == nil {
		t.Fatal("expected discriminator error")
	}

	data = buildMatchStateAccountData("18237038", false, 0)
	data[40] = 33
	if _, err := DecodeMatchState(pubkey, data); err == nil {
		t.Fatal("expected match id length error")
	}

	data = buildMatchStateAccountData("18237038", false, 0)
	data[41] = 2
	if _, err := DecodeMatchState(pubkey, data); err == nil {
		t.Fatal("expected closed flag error")
	}
}

func TestCloseMatchSkipsAlreadyClosedState(t *testing.T) {
	const matchID = "18237038"
	mock := newMockRPC()
	defer mock.Close()

	data := buildMatchStateAccountData(matchID, true, 1784063054)
	mock.getAccountInfo = func(string) (json.RawMessage, error) {
		payload, _ := json.Marshal(map[string]any{
			"context": map[string]any{"slot": 100},
			"value": map[string]any{
				"data":       encodeAccountData(data),
				"executable": false,
				"lamports":   1,
				"owner":      testProgramID,
			},
		})
		return payload, nil
	}
	sendCalls := 0
	mock.sendTransaction = func() (json.RawMessage, error) {
		sendCalls++
		return json.RawMessage(`"` + testSignature() + `"`), nil
	}

	client := testClient(t, mock)
	_, err := client.CloseMatch(context.Background(), solana.NewWallet().PrivateKey, matchID)
	if !errors.Is(err, ErrMatchAlreadyClosed) {
		t.Fatalf("err = %v, want ErrMatchAlreadyClosed", err)
	}
	if sendCalls != 0 {
		t.Fatalf("send calls = %d, want 0", sendCalls)
	}
}

func TestGetMatchStateValidatesOwnerAndMatchID(t *testing.T) {
	const matchID = "18237038"
	mock := newMockRPC()
	defer mock.Close()
	client := testClient(t, mock)

	data := buildMatchStateAccountData(matchID, false, 0)
	mock.getAccountInfo = func(string) (json.RawMessage, error) {
		payload, _ := json.Marshal(map[string]any{
			"context": map[string]any{"slot": 100},
			"value": map[string]any{
				"data":       encodeAccountData(data),
				"executable": false,
				"lamports":   1,
				"owner":      testMint,
			},
		})
		return payload, nil
	}
	if _, _, err := client.GetMatchState(context.Background(), matchID); err == nil {
		t.Fatal("expected owner error")
	}

	data = buildMatchStateAccountData("17952170", false, 0)
	mock.getAccountInfo = func(string) (json.RawMessage, error) {
		payload, _ := json.Marshal(map[string]any{
			"context": map[string]any{"slot": 100},
			"value": map[string]any{
				"data":       encodeAccountData(data),
				"executable": false,
				"lamports":   1,
				"owner":      testProgramID,
			},
		})
		return payload, nil
	}
	if _, _, err := client.GetMatchState(context.Background(), matchID); err == nil {
		t.Fatal("expected match id mismatch")
	}
}
