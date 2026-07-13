package cache

import "time"

// MatchDuration is the typical elapsed window after kickoff before we start probing TxLINE
// for a missed terminal update. It must never be used as authoritative evidence that a
// fixture is final.
const MatchDuration = 105 * time.Minute

// FinalVerificationEligible reports whether a non-final match is old enough that we should
// verify its latest state via TxLINE snapshot in case an SSE final event was missed.
func FinalVerificationEligible(match Match, now time.Time) bool {
	if match.IsFinal {
		return false
	}
	if match.StartTime <= 0 {
		return false
	}
	return now.UnixMilli()-match.StartTime >= MatchDuration.Milliseconds()
}
