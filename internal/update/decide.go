package update

import "time"

// Offer reports whether latest should be offered to a build running current.
// Dev and snapshot builds are never offered an update.
func Offer(current string, latest Version) bool {
	cur, ok := ParseVersion(current)
	if !ok || cur.Pre != "" {
		return false
	}
	return cur.Less(latest)
}

// Due reports whether an automatic check should run now.
func Due(last, now time.Time, every time.Duration) bool {
	if last.IsZero() || last.After(now) {
		return true
	}
	return now.Sub(last) >= every
}
