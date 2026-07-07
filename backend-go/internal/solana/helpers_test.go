package solana

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestIdempotentHelpers(t *testing.T) {
	if !isIdempotentSettleError("InstructionError InvalidStatus") {
		t.Fatal("expected settle idempotent match")
	}
	if !isIdempotentSettleError(map[string]any{"Custom": 6001}) {
		t.Fatal("expected custom code match")
	}
	if isIdempotentSettleError("other failure") {
		t.Fatal("unexpected match")
	}

	err := errors.New("InvalidStatus already in use")
	if !isIdempotentSendError(err) {
		t.Fatal("expected send idempotent match")
	}
	if isIdempotentSendError(nil) {
		t.Fatal("nil should be false")
	}

	if !stringContains("abc", "b") || stringContains("abc", "z") {
		t.Fatal("stringContains mismatch")
	}
	if !containsAny("x6001y", "6001") || containsAny("", "") {
		t.Fatal("containsAny mismatch")
	}
	if indexOf("hello", "ll") != 2 || indexOf("hello", "zz") != -1 {
		t.Fatal("indexOf mismatch")
	}
}

func TestWaitForSignature(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	client := rpc.New(mock.URL())

	var sig solana.Signature
	sig[0] = 7
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := waitForSignature(ctx, client, sig); err != nil {
		t.Fatalf("waitForSignature: %v", err)
	}

	mock.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[{"slot":100,"err":{"InstructionError":[0,"Custom"]},"confirmationStatus":"confirmed"}]}`)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	if err := waitForSignature(ctx2, client, sig); err == nil {
		t.Fatal("expected transaction failure")
	}

	mock.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[null]}`)
	}
	ctx3, cancel3 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel3()
	if err := waitForSignature(ctx3, client, sig); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}