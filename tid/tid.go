package tid

import (
	"errors"
	"regexp"
	"sync"
	"time"
)

var tidRegex = regexp.MustCompile(`^[234567abcdefghij][234567abcdefghijklmnopqrstuvwxyz]{12}$`)

func padLeft(str string, length int, padChar rune) string {
	for len(str) < length {
		str = string(padChar) + str
	}

	return str
}

func createRaw(timestamp, clockId int64) string {
	return padLeft(b32Encode(timestamp), 11, '2') + padLeft(b32Encode(clockId), 2, '2')
}

// Creates a TID string from a timestamp (in milliseconds) and clock ID value.
func Create(timestamp, clockId int64) (string, error) {
	if timestamp < 0 {
		return "", errors.New("timestamp must be positive")
	}

	if clockId < 0 {
		return "", errors.New("clockId must be positive")
	}

	return createRaw(timestamp, clockId), nil
}

// Parses a TID string into a timestamp (in milliseconds) and clock ID value.
func Parse(s string) (int64, int64, error) {
	if err := Validate(s); err != nil {
		return 0, 0, err
	}

	timestamp := b32Decode(s[0:11])
	clockId := b32Decode(s[11:13])

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
	id   int64
	mtx  sync.Mutex
	last int64
}

func NewClock(id int64) Clock {
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

	return createRaw(now, c.id)
}
