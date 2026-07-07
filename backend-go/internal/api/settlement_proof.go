package api

import (
	"net/http"
	"strings"

	"github.com/matchlock/backend-go/internal/keeper"
)

func (h *handler) getWagerSettlementProof(w http.ResponseWriter, r *http.Request) {
	pubkeyRaw := strings.TrimSpace(r.PathValue("pubkey"))
	if pubkeyRaw == "" {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", "wager pubkey is required")
		return
	}
	pubkey, err := parseWagerPubkey(pubkeyRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", "invalid wager pubkey")
		return
	}

	wager, err := h.wagers.GetWager(r.Context(), pubkey)
	if err != nil {
		if isWagerMissing(err) {
			writeError(w, http.StatusNotFound, "WAGER_NOT_FOUND", "wager not found or already settled")
			return
		}
		writeError(w, http.StatusBadGateway, "RPC_ERROR", "failed to load wager")
		return
	}

	if h.txlineData == nil || h.solana == nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "settlement proof service unavailable")
		return
	}

	builder := keeper.ProofBuilder{
		Cache:  h.cache,
		Txline: h.txlineData,
		Solana: h.solana,
	}
	proof, err := builder.BuildForWager(r.Context(), wager)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not final"):
			writeError(w, http.StatusConflict, "MATCH_NOT_FINAL", "match is not final yet")
		case strings.Contains(msg, "not matched"):
			writeError(w, http.StatusConflict, "INVALID_STATUS", "wager is not eligible for settlement")
		default:
			writeError(w, http.StatusBadGateway, "PROOF_UNAVAILABLE", "could not build settlement proof")
		}
		return
	}

	writeJSON(w, http.StatusOK, proof)
}