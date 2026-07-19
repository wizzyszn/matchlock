package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/matchlock/backend-go/internal/txline"
)

func (h *handler) getFixtureValidation(w http.ResponseWriter, r *http.Request) {
	fixtureIDRaw := strings.TrimSpace(r.URL.Query().Get("fixtureId"))
	if fixtureIDRaw == "" {
		writeError(w, http.StatusBadRequest, "INVALID_FIXTURE_ID", "fixtureId query parameter is required")
		return
	}

	fixtureID, err := strconv.ParseInt(fixtureIDRaw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FIXTURE_ID", "fixtureId must be a valid integer")
		return
	}

	var timestamp *int64
	if tsRaw := strings.TrimSpace(r.URL.Query().Get("timestamp")); tsRaw != "" {
		ts, err := strconv.ParseInt(tsRaw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_TIMESTAMP", "timestamp must be a valid integer")
			return
		}
		timestamp = &ts
	}

	txlineClient, ok := h.txlineData.(*txline.Client)
	if !ok {
		writeError(w, http.StatusInternalServerError, "NOT_SUPPORTED", "fixture validation not supported")
		return
	}

	result, err := txlineClient.FetchFixtureValidation(r.Context(), fixtureID, timestamp)
	if err != nil {
		writeError(w, http.StatusBadGateway, "TXLINE_ERROR", "failed to fetch fixture validation: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
