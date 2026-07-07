package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/matchlock/backend-go/internal/auth"
	"github.com/matchlock/backend-go/internal/db"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

func (h *handler) postChallengeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	var body struct {
		RecipientEmail string `json:"recipient_email"`
		WagerPubkey    string `json:"wager_pubkey"`
		MatchID        string `json:"match_id"`
		MakerSide      string `json:"maker_side"`
		Stake          uint64 `json:"stake"`
		HomeTeam       string `json:"home_team"`
		AwayTeam       string `json:"away_team"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	invite, err := h.auth.CreateWagerInvite(r.Context(), user, auth.CreateInviteInput{
		RecipientEmail: body.RecipientEmail,
		WagerPubkey:    body.WagerPubkey,
		MatchID:        body.MatchID,
		MakerSide:      body.MakerSide,
		Stake:          body.Stake,
		HomeTeam:       body.HomeTeam,
		AwayTeam:       body.AwayTeam,
	})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidInvite) {
			writeError(w, http.StatusBadRequest, "INVALID_INVITE", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INVITE_ERROR", "could not create invite")
		return
	}
	writeJSON(w, http.StatusCreated, invite)
}

func (h *handler) listChallengeInvites(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	invites, err := h.auth.ListInvites(r.Context(), user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INVITE_ERROR", "could not list invites")
		return
	}
	if invites == nil {
		invites = []auth.InviteView{}
	}

	for i, inv := range invites {
		if inv.WagerPubkey == "" {
			continue
		}
		pk, parseErr := solana.PublicKeyFromBase58(inv.WagerPubkey)
		if parseErr != nil {
			continue
		}
		wager, lookupErr := h.wagers.GetWager(r.Context(), pk)
		if lookupErr != nil {
			continue
		}
		if wager.Status == chainsol.WagerStatusMatched || wager.Status == chainsol.WagerStatusSettled {
			invites[i].Status = "accepted"
		}
	}

	writeJSON(w, http.StatusOK, invites)
}

func (h *handler) getChallengeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid invite id")
		return
	}
	invite, err := h.auth.GetInvite(r.Context(), user, id)
	if err != nil {
		if errors.Is(err, auth.ErrInviteNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "invite not found")
			return
		}
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "not allowed to view this invite")
			return
		}
		writeError(w, http.StatusInternalServerError, "INVITE_ERROR", "could not load invite")
		return
	}
	writeJSON(w, http.StatusOK, invite)
}

func (h *handler) patchChallengeInvite(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid invite id")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	status := db.WagerInviteStatus(strings.TrimSpace(body.Status))
	invite, err := h.auth.UpdateInviteStatus(r.Context(), user, id, status)
	if err != nil {
		if errors.Is(err, auth.ErrInviteNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "invite not found")
			return
		}
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "only the recipient can respond")
			return
		}
		if errors.Is(err, auth.ErrInvalidInvite) {
			writeError(w, http.StatusBadRequest, "INVALID_INVITE", "invite cannot be updated")
			return
		}
		writeError(w, http.StatusInternalServerError, "INVITE_ERROR", "could not update invite")
		return
	}
	writeJSON(w, http.StatusOK, invite)
}

func (h *handler) postChallengeInviteWager(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ID", "invalid invite id")
		return
	}
	var body struct {
		WagerPubkey string `json:"wager_pubkey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	wagerPubkey := strings.TrimSpace(body.WagerPubkey)
	if wagerPubkey == "" {
		writeError(w, http.StatusBadRequest, "INVALID_WAGER", "wager_pubkey required")
		return
	}
	pk, err := solana.PublicKeyFromBase58(wagerPubkey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_WAGER", "invalid wager pubkey")
		return
	}
	wager, err := h.wagers.GetWager(r.Context(), pk)
	if err != nil {
		writeError(w, http.StatusBadRequest, "WAGER_NOT_FOUND", "wager not found on chain")
		return
	}
	if err := h.auth.AssertUserOwnsWallet(r.Context(), user.ID, wager.Maker.String()); err != nil {
		if errors.Is(err, auth.ErrWalletNotLinked) {
			writeError(w, http.StatusForbidden, "WALLET_NOT_LINKED", "wager maker must be a wallet linked to your account")
			return
		}
		if errors.Is(err, auth.ErrWalletOwnedByOther) {
			writeError(w, http.StatusForbidden, "WALLET_OWNED_BY_OTHER", "wager maker wallet belongs to another account")
			return
		}
		writeError(w, http.StatusForbidden, "WAGER_MAKER_MISMATCH", "wager maker does not match your linked wallet")
		return
	}
	invite, err := h.auth.AttachWagerToInvite(r.Context(), user, id, wagerPubkey)
	if err != nil {
		if errors.Is(err, auth.ErrInviteNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "invite not found")
			return
		}
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "only the maker can attach a wager")
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_INVITE", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, invite)
}
