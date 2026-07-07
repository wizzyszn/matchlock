package cache

import "time"

// MatchDuration is the typical elapsed window before a soccer fixture is treated as finished
// when TxLINE still reports a non-terminal gameState (e.g. "scheduled").
const MatchDuration = 105 * time.Minute

// InferFinalState marks fixtures as final when kickoff was long enough ago and scores exist.
func InferFinalState(match Match, now time.Time) Match {
	out := match
	if out.IsFinal {
		return out
	}
	if out.StartTime <= 0 {
		return out
	}
	if now.UnixMilli()-out.StartTime < MatchDuration.Milliseconds() {
		return out
	}

	out.IsFinal = true
	if out.FinalSource != FinalSourceTxline {
		out.FinalSource = FinalSourceInferred
	}
	if out.GameState == "" || out.GameState == "scheduled" {
		out.GameState = "FT"
	}
	if out.FinalizedAt == nil {
		ts := now.UTC()
		out.FinalizedAt = &ts
	}
	return out
}