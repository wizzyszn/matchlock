package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/matchlock/backend-go/internal/auth"
)

func clientIP(r *http.Request) string {
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (h *handler) postMagicLink(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "AUTH_DISABLED", "auth not configured")
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	if err := h.auth.RequestMagicLink(r.Context(), body.Email); err != nil {
		if errors.Is(err, auth.ErrRateLimited) {
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "please wait a minute before requesting another sign-in link (max 8 per hour)")
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_EMAIL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) getVerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "AUTH_DISABLED", "auth not configured")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	sess, err := h.auth.VerifyMagicLink(r.Context(), token, r.UserAgent(), clientIP(r))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "link expired or already used")
			return
		}
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not verify login")
		return
	}
	auth.SetAccessCookie(w, h.tokenCfg, sess.AccessToken, sess.AccessExpiry)
	auth.SetRefreshCookie(w, h.tokenCfg, sess.RefreshRaw, sess.RefreshExpiry)
	profile, err := h.auth.GetProfile(r.Context(), sess.User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not load profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handler) postRefresh(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "AUTH_DISABLED", "auth not configured")
		return
	}
	refresh, ok := auth.ReadCookie(r, auth.RefreshCookieName)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "refresh token missing")
		return
	}
	sess, err := h.auth.RefreshSession(r.Context(), refresh, r.UserAgent(), clientIP(r))
	if err != nil {
		auth.ClearAuthCookies(w, h.tokenCfg)
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session expired; sign in again")
		return
	}
	auth.SetAccessCookie(w, h.tokenCfg, sess.AccessToken, sess.AccessExpiry)
	auth.SetRefreshCookie(w, h.tokenCfg, sess.RefreshRaw, sess.RefreshExpiry)
	profile, err := h.auth.GetProfile(r.Context(), sess.User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not load profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handler) postLogout(w http.ResponseWriter, r *http.Request) {
	if h.auth != nil {
		if refresh, ok := auth.ReadCookie(r, auth.RefreshCookieName); ok {
			_ = h.auth.Logout(r.Context(), refresh)
		}
	}
	auth.ClearAuthCookies(w, h.tokenCfg)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) patchMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	profile, err := h.auth.UpdateDisplayName(r.Context(), user.ID, body.DisplayName)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidDisplayName) {
			writeError(w, http.StatusBadRequest, "INVALID_DISPLAY_NAME", "username must be 3-32 characters (letters, numbers, underscore)")
			return
		}
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not update profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handler) getMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	profile, err := h.auth.GetProfile(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not load profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handler) getWalletCheck(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	pubkey := strings.TrimSpace(r.URL.Query().Get("pubkey"))
	view, err := h.auth.CheckWalletBinding(r.Context(), user.ID, pubkey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PUBKEY", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *handler) postWalletLinkChallenge(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	var body struct {
		Pubkey string `json:"pubkey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	message, err := h.auth.CreateWalletLinkChallenge(r.Context(), user.ID, body.Pubkey)
	if err != nil {
		if errors.Is(err, auth.ErrWalletOwnedByOther) {
			writeError(w, http.StatusConflict, "WALLET_OWNED_BY_OTHER", "this wallet is already linked to another Matchlock account")
			return
		}
		writeError(w, http.StatusBadRequest, "CHALLENGE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": message,
		"pubkey":  strings.TrimSpace(body.Pubkey),
	})
}

func (h *handler) postWalletLink(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	var body struct {
		Pubkey    string `json:"pubkey"`
		Message   string `json:"message"`
		Signature string `json:"signature"`
		Label     string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	view, err := h.auth.LinkWallet(r.Context(), user.ID, body.Pubkey, body.Message, body.Signature, body.Label)
	if err != nil {
		if errors.Is(err, auth.ErrWalletOwnedByOther) {
			writeError(w, http.StatusConflict, "WALLET_OWNED_BY_OTHER", "this wallet is already linked to another Matchlock account")
			return
		}
		writeError(w, http.StatusBadRequest, "WALLET_LINK_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *handler) postWalletPrimary(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	pubkey := strings.TrimSpace(r.PathValue("pubkey"))
	if err := h.auth.SetPrimaryWallet(r.Context(), user.ID, pubkey); err != nil {
		if errors.Is(err, auth.ErrWalletNotLinked) {
			writeError(w, http.StatusNotFound, "WALLET_NOT_FOUND", "wallet not linked")
			return
		}
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not update primary wallet")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) deleteWallet(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	pubkey := strings.TrimSpace(r.PathValue("pubkey"))
	if err := h.auth.UnlinkWallet(r.Context(), user.ID, pubkey); err != nil {
		if errors.Is(err, auth.ErrWalletNotLinked) {
			writeError(w, http.StatusNotFound, "WALLET_NOT_FOUND", "wallet not linked")
			return
		}
		writeError(w, http.StatusInternalServerError, "AUTH_ERROR", "could not unlink wallet")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) getUserLookup(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	_ = user
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	out, err := h.auth.LookupUserByEmail(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_EMAIL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}