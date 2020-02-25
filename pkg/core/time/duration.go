package time

import (
	"time"
)

func ToMilliseconds(d time.Duration) int64 {
	return int64(d / time.Millisecond)
}
