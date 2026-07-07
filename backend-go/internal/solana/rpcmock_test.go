package solana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/gagliardetto/solana-go"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type mockRPC struct {
	server *httptest.Server
	mu     sync.Mutex
	slot   uint64

	getAccountInfo      func(pubkey string) (json.RawMessage, error)
	getProgramAccounts  func() json.RawMessage
	simulateTransaction func() (json.RawMessage, error)
	sendTransaction     func() (json.RawMessage, error)
	signatureStatuses    func() json.RawMessage
	getLatestBlockhashErr error
}

func testBlockhash() string {
	var h solana.Hash
	h[0] = 9
	return h.String()
}

func testSignature() string {
	var sig solana.Signature
	sig[0] = 7
	return sig.String()
}

func newMockRPC() *mockRPC {
	m := &mockRPC{slot: 100}
	m.server = httptest.NewServer(http.HandlerFunc(m.handle))
	m.getAccountInfo = func(pubkey string) (json.RawMessage, error) {
		return json.RawMessage(`{"context":{"slot":100},"value":null}`), nil
	}
	m.getProgramAccounts = func() json.RawMessage {
		return json.RawMessage(`[]`)
	}
	m.simulateTransaction = func() (json.RawMessage, error) {
		return json.RawMessage(`{"context":{"slot":100},"value":{"err":null,"logs":[]}}`), nil
	}
	m.sendTransaction = func() (json.RawMessage, error) {
		return json.RawMessage(`"` + testSignature() + `"`), nil
	}
	m.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[{"slot":100,"confirmations":1,"err":null,"confirmationStatus":"confirmed"}]}`)
	}
	return m
}

func (m *mockRPC) URL() string { return m.server.URL }

func (m *mockRPC) Close() { m.server.Close() }

func (m *mockRPC) handle(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var result any
	var rpcErr *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	switch req.Method {
	case "getSlot":
		result = m.slot
	case "getAccountInfo":
		var params []json.RawMessage
		_ = json.Unmarshal(req.Params, &params)
		var pubkey string
		if len(params) > 0 {
			_ = json.Unmarshal(params[0], &pubkey)
		}
		raw, err := m.getAccountInfo(pubkey)
		if err != nil {
			rpcErr = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32000, Message: err.Error()}
		} else {
			_ = json.Unmarshal(raw, &result)
		}
	case "getProgramAccounts":
		var accounts []json.RawMessage
		_ = json.Unmarshal(m.getProgramAccounts(), &accounts)
		result = accounts
	case "getLatestBlockhash":
		if m.getLatestBlockhashErr != nil {
			rpcErr = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32000, Message: m.getLatestBlockhashErr.Error()}
			break
		}
		result = map[string]any{
			"context": map[string]any{"slot": m.slot},
			"value": map[string]any{
				"blockhash":            testBlockhash(),
				"lastValidBlockHeight": 1000,
			},
		}
	case "simulateTransaction":
		raw, err := m.simulateTransaction()
		if err != nil {
			rpcErr = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32000, Message: err.Error()}
		} else {
			_ = json.Unmarshal(raw, &result)
		}
	case "sendTransaction":
		raw, err := m.sendTransaction()
		if err != nil {
			rpcErr = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32000, Message: err.Error()}
		} else {
			_ = json.Unmarshal(raw, &result)
		}
	case "getSignatureStatuses":
		_ = json.Unmarshal(m.signatureStatuses(), &result)
	default:
		rpcErr = &struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{Code: -32601, Message: "method not found: " + req.Method}
	}

	resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
	if rpcErr != nil {
		resp["error"] = rpcErr
	} else {
		resp["result"] = result
	}
	_ = json.NewEncoder(w).Encode(resp)
}