package tid

import (
	"errors"
	"regexp"
	"sync"
	"time"
)

var tidRegex = regexp.MustCompile(`^[234567abcdefghij][234567abcdefghijklmnopqrstuvwxyz]{12}$`)

// Creates a TID string from a timestamp (in milliseconds) and clock ID value.
func Create(timestamp int64, clockId uint) string {
	v := (uint64(timestamp&0x1F_FFFF_FFFF_FFFF) << 10) | uint64(clockId&0x3FF)
	return b32Encode(v)
}

// Parses a TID string into a timestamp (in milliseconds) and clock ID value.
func Parse(s string) (timestamp, clockId uint, err error) {
	if err = Validate(s); err != nil {
		return 0, 0, err
	}
	timestamp = b32Decode(s[0:11])
	clockId = b32Decode(s[11:13])
	return timestamp, clockId, nil
}

// Validates a TID string.
func Validate(s string) error {
	if len(s) != 13 {
		return errors.New("invalid tid length")
	}
	if !tidRegex.MatchString(s) {
		return errors.New("invalid tid format")
	}
	return nil
}

// TID generator, which keeps state to ensure TID values always monotonically increase.
type Clock struct {
	id   uint
	mtx  sync.Mutex
	last int64
}

func NewClock(id uint) Clock {
	return Clock{id: id}
}

// Returns a TID string based on current time.
func (c *Clock) Now() string {
	now := time.Now().UTC().UnixMicro()
	c.mtx.Lock()
	if now <= c.last {
		now = c.last + 1
	}
	c.last = now
	c.mtx.Unlock()
	return Create(now, c.id)
}
