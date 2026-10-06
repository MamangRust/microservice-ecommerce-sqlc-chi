package adapter

import "time"

// ParseTimePtr parses a proto timestamp string into a *time.Time. The services
// serialize timestamps as RFC3339, so that is the only layout accepted here.
// An empty or unparseable value yields nil, which the callers treat as "no
// timestamp" (for example when materializing stats events).
func ParseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
