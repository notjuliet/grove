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

func createRaw(timestamp, clockid int64) string {
	return padLeft(b32Encode(timestamp), 11, '2') + padLeft(b32Encode(clockid), 2, '2')
}

func Create(timestamp, clockid int64) (string, error) {
	if timestamp < 0 {
		return "", errors.New("timestamp must be positive")
	}

	if clockid < 0 {
		return "", errors.New("clockid must be positive")
	}

	return createRaw(timestamp, clockid), nil
}

func Parse(s string) (int64, int64, error) {
	if err := Validate(s); err != nil {
		return 0, 0, err
	}

	timestamp := b32Decode(s[0:11])
	clockid := b32Decode(s[11:13])

	return timestamp, clockid, nil
}

func Validate(s string) error {
	if len(s) != 13 {
		return errors.New("invalid tid length")
	}

	if !tidRegex.MatchString(s) {
		return errors.New("invalid tid format")
	}

	return nil
}

type Clock struct {
	id   int64
	mtx  sync.Mutex
	last int64
}

func NewClock(id int64) Clock {
	return Clock{id: id}
}

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
