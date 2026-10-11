package scrapbox

import "time"

// Timestamp is UNIX seconds returned by the API, kept as a number in JSON.
type Timestamp int64

// Time converts t to time.Time.
func (t Timestamp) Time() time.Time { return time.Unix(int64(t), 0) }
