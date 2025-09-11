package quartz

import (
	"time"
)

// Sep is the serialization delimiter; the default is a double colon.
var Sep = "::"

// NowNano returns the current Unix time in nanoseconds.
func NowNano() int64 {
	return time.Now().UnixNano()
}
