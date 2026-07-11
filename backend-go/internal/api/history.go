package api

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
)

const (
	historySettlementSettled   = "settled"
	historySettlementUnsettled = "unsettled"

	historyOutcomeWon  = "won"
	historyOutcomeLost = "lost"
	historyOutcomeVoid = "void"
)

type wagerHistoryFilter struct {
	Wallet           string
	SettlementStatus string
	Outcome          string
	From             *int64
	To               *int64
	Offset           int
	Limit            int
}

func (h *handler) listWagerHistory(w http.ResponseWriter, r *http.Request) {
	filter, err := parseWagerHistoryFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	wagers, err := h.wagers.ListWagers(r.Context(), chainsol.WagerFilter{Wallet: filter.Wallet})
	if err != nil {
		writeError(w, http.StatusBadGateway, "RPC_ERROR", "failed to list wager history")
		return
	}

	matches, err := h.cache.ListMatches(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CACHE_ERROR", "failed to load match history")
		return
	}

	matchMap := make(map[string]cache.Match, len(matches))
	for _, match := range matches {
		matchMap[match.MatchID] = match
	}

	out := make([]WagerHistoryEntryView, 0, len(wagers))
	for _, wager := range wagers {
		entry, ok := wagerHistoryEntryFromData(wager, matchMap[wager.MatchIDString()], filter.Wallet)
		if !ok {
			continue
		}
		if !historyEntryMatchesFilter(entry, filter) {
			continue
		}
		out = append(out, entry)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EventTime != out[j].EventTime {
			return out[i].EventTime > out[j].EventTime
		}
		return out[i].Wager.Pubkey > out[j].Wager.Pubkey
	})

	total := len(out)
	start := filter.Offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}

	writeJSON(w, http.StatusOK, WagerHistoryPageView{
		Entries: out[start:end],
		Total:   total,
		Offset:  filter.Offset,
		Limit:   filter.Limit,
		HasMore: end < total,
	})
}

func parseWagerHistoryFilter(r *http.Request) (wagerHistoryFilter, error) {
	query := r.URL.Query()
	filter := wagerHistoryFilter{
		Wallet: strings.TrimSpace(query.Get("wallet")),
	}
	if filter.Wallet == "" {
		return wagerHistoryFilter{}, errors.New("wallet is required")
	}
	if _, err := solana.PublicKeyFromBase58(filter.Wallet); err != nil {
		return wagerHistoryFilter{}, errors.New("wallet must be a valid base58 pubkey")
	}

	settlementStatus := strings.ToLower(strings.TrimSpace(query.Get("settlement_status")))
	switch settlementStatus {
	case "":
	case historySettlementSettled, historySettlementUnsettled:
		filter.SettlementStatus = settlementStatus
	default:
		return wagerHistoryFilter{}, errors.New("settlement_status must be settled or unsettled")
	}

	outcome := strings.ToLower(strings.TrimSpace(query.Get("outcome")))
	switch outcome {
	case "":
	case historyOutcomeWon, historyOutcomeLost, historyOutcomeVoid:
		filter.Outcome = outcome
	default:
		return wagerHistoryFilter{}, errors.New("outcome must be won, lost, or void")
	}

	from, err := parseOptionalUnixMillis(query.Get("from"))
	if err != nil {
		return wagerHistoryFilter{}, errors.New("from must be a valid unix timestamp in milliseconds")
	}
	to, err := parseOptionalUnixMillis(query.Get("to"))
	if err != nil {
		return wagerHistoryFilter{}, errors.New("to must be a valid unix timestamp in milliseconds")
	}
	if from != nil && to != nil && *from > *to {
		return wagerHistoryFilter{}, errors.New("from must be less than or equal to to")
	}
	filter.From = from
	filter.To = to

	offset, err := parseOptionalInt(query.Get("offset"))
	if err != nil || offset < 0 {
		return wagerHistoryFilter{}, errors.New("offset must be a non-negative integer")
	}
	limit, err := parseOptionalInt(query.Get("limit"))
	if err != nil || limit < 0 {
		return wagerHistoryFilter{}, errors.New("limit must be a non-negative integer")
	}
	if limit == 0 {
		limit = 25
	}
	if limit > 100 {
		return wagerHistoryFilter{}, errors.New("limit must be less than or equal to 100")
	}
	filter.Offset = offset
	filter.Limit = limit

	return filter, nil
}

func parseOptionalUnixMillis(raw string) (*int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func wagerHistoryEntryFromData(
	wager chainsol.Wager,
	match cache.Match,
	wallet string,
) (WagerHistoryEntryView, bool) {
	isMaker := wager.Maker.String() == wallet
	isTaker := wager.HasCounterparty() && wager.Taker.String() == wallet
	if !isMaker && !isTaker {
		return WagerHistoryEntryView{}, false
	}

	backedSide := wager.TakerSide
	opponent := wager.Maker.String()
	if isMaker {
		backedSide = wager.MakerSide
		if wager.HasCounterparty() {
			opponent = wager.Taker.String()
		} else {
			opponent = ""
		}
	}

	entry := WagerHistoryEntryView{
		Wager:            wagerViewFromChain(wager),
		SettlementStatus: historySettlementStatus(wager.Status),
		BackedSide:       chainsol.SideName(backedSide),
		Opponent:         opponent,
		IsMaker:          isMaker,
	}
	if match.MatchID != "" {
		matchView := matchViewFromCache(match)
		entry.Match = &matchView
		entry.EventTime = match.StartTime
	}
	if outcome, ok := historyOutcomeFromData(wager, match, backedSide); ok {
		entry.Outcome = outcome
	}
	return entry, true
}

func historySettlementStatus(status uint8) string {
	if status == chainsol.WagerStatusSettled || status == chainsol.WagerStatusCancelled {
		return historySettlementSettled
	}
	return historySettlementUnsettled
}

func historyOutcomeFromData(wager chainsol.Wager, match cache.Match, backedSide uint8) (string, bool) {
	if wager.Status == chainsol.WagerStatusCancelled {
		return historyOutcomeVoid, true
	}
	if wager.Status != chainsol.WagerStatusSettled {
		return "", false
	}
	winningSide, ok := winningSideFromMatch(match)
	if !ok {
		return "", false
	}
	if backedSide == winningSide {
		return historyOutcomeWon, true
	}
	return historyOutcomeLost, true
}

func winningSideFromMatch(match cache.Match) (uint8, bool) {
	if !match.IsFinal || match.HomeGoals == nil || match.AwayGoals == nil {
		return 0, false
	}
	switch {
	case *match.HomeGoals > *match.AwayGoals:
		return chainsol.SideHome, true
	case *match.AwayGoals > *match.HomeGoals:
		return chainsol.SideAway, true
	default:
		return chainsol.SideDraw, true
	}
}

func historyEntryMatchesFilter(entry WagerHistoryEntryView, filter wagerHistoryFilter) bool {
	if filter.SettlementStatus != "" && entry.SettlementStatus != filter.SettlementStatus {
		return false
	}
	if filter.Outcome != "" && entry.Outcome != filter.Outcome {
		return false
	}
	if filter.From != nil {
		if entry.EventTime == 0 || entry.EventTime < *filter.From {
			return false
		}
	}
	if filter.To != nil {
		if entry.EventTime == 0 || entry.EventTime > *filter.To {
			return false
		}
	}
	return true
}
