package solana

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
)

func TestGetWagerRPCError(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.getAccountInfo = func(string) (json.RawMessage, error) {
		return nil, errors.New("rpc unavailable")
	}
	client := testClient(t, mock)
	_, err := client.GetWager(context.Background(), solana.NewWallet().PublicKey())
	if err == nil {
		t.Fatal("expected rpc error")
	}
}

func TestListWagersFiltersMismatch(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	pubkey := solana.NewWallet().PublicKey()
	data := buildWagerAccountData(t, "11111111", WagerStatusOpen, 1)
	mock.getProgramAccounts = func() json.RawMessage {
		payload, _ := json.Marshal([]map[string]any{{
			"pubkey": pubkey.String(),
			"account": map[string]any{
				"data": encodeAccountData(data), "executable": false,
				"lamports": 1, "owner": testProgramID, "rentEpoch": 0,
			},
		}})
		return payload
	}
	client := testClient(t, mock)
	status := WagerStatusMatched
	wagers, err := client.ListWagers(context.Background(), WagerFilter{
		Status:  &status,
		MatchID: "99999999",
	})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 0 {
		t.Fatalf("len = %d", len(wagers))
	}
}

func TestSettleWagerGetBlockhashError(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.getLatestBlockhashErr = errors.New("blockhash unavailable")
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil || !strings.Contains(err.Error(), "blockhash") {
		t.Fatalf("err = %v", err)
	}
}

func TestSettleWagerSimulationFailure(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.simulateTransaction = func() (json.RawMessage, error) {
		return json.RawMessage(`{"context":{"slot":100},"value":{"err":"boom","logs":["failed"]}}`), nil
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil || errors.Is(err, ErrAlreadySettled) {
		t.Fatalf("err = %v", err)
	}
}

func TestSettleWagerSimulateRPCError(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.simulateTransaction = func() (json.RawMessage, error) {
		return nil, errors.New("rpc simulate down")
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil {
		t.Fatal("expected simulate rpc error")
	}
}

func TestSettleWagerSendRPCError(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.sendTransaction = func() (json.RawMessage, error) {
		return nil, errors.New("network down")
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil {
		t.Fatal("expected send rpc error")
	}
}

func TestSettleWagerConfirmFailureReturnsSig(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[null]}`)
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	sig, err := client.SettleWager(ctx, SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil || sig.IsZero() {
		t.Fatalf("sig=%v err=%v", sig, err)
	}
}

func TestClientPingRPCError(t *testing.T) {
	mock := newMockRPC()
	client := testClient(t, mock)
	mock.server.Close()
	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("expected ping error")
	}
}

func TestListWagersRPCError(t *testing.T) {
	mock := newMockRPC()
	client := testClient(t, mock)
	mock.server.Close()
	if _, err := client.ListWagers(context.Background(), WagerFilter{}); err == nil {
		t.Fatal("expected rpc error")
	}
}

func TestGetWagerDecodeError(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	pubkey := solana.NewWallet().PublicKey()
	mock.getAccountInfo = func(string) (json.RawMessage, error) {
		payload, _ := json.Marshal(map[string]any{
			"context": map[string]any{"slot": 100},
			"value": map[string]any{
				"data": encodeAccountData([]byte{1, 2, 3}),
				"executable": false, "lamports": 1, "owner": testProgramID,
			},
		})
		return payload, nil
	}
	client := testClient(t, mock)
	if _, err := client.GetWager(context.Background(), pubkey); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestIsIdempotentHelpersExtended(t *testing.T) {
	if !isIdempotentSettleError("AccountNotFound") {
		t.Fatal("expected AccountNotFound")
	}
	if !isIdempotentSendError(errors.New("6001")) {
		t.Fatal("expected 6001 in send error")
	}
	if indexOf("abc", "") != 0 {
		t.Fatal("empty substring")
	}
}

func matchedWager(maker solana.PublicKey) Wager {
	w := Wager{
		Pubkey: solana.NewWallet().PublicKey(), Maker: maker,
		Taker: solana.NewWallet().PublicKey(), MatchIDLen: 8,
		MakerSide: SideHome, TakerSide: SideAway, Status: WagerStatusMatched,
	}
	copy(w.MatchID[:], []byte("17952170"))
	return w
}