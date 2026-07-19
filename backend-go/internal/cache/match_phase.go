package cache

import "time"

// MatchDuration is the typical elapsed window after kickoff before we start probing TxLINE
// for a missed terminal update. It must never be used as authoritative evidence that a
// fixture is final.
const MatchDuration = 105 * time.Minute

// MaxLiveStatusAge bounds how long a fixture may be presented as live without a
// verified terminal update. Crossing this boundary does not authorize settlement.
const MaxLiveStatusAge = 4 * time.Hour

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

// LiveStatusExpired reports that a non-final fixture is too old to be represented
// as live. Settlement still requires a TxLINE-verified final snapshot and proof.
func LiveStatusExpired(match Match, now time.Time) bool {
	if match.IsFinal || match.StartTime <= 0 {
		return false
	}
	return now.UnixMilli()-match.StartTime >= MaxLiveStatusAge.Milliseconds()
}
