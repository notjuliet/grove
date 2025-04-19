package tid

import (
	"testing"
)

func TestCreate(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		s, err := Create(1234567890, 0)
		if err != nil {
			t.Fatal(err)
		}

		if s != "222236tg2qm22" {
			t.Fatal("invalid tid")
		}
	})

	t.Run("negative", func(t *testing.T) {
		_, err := Create(-1, 0)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		timestamp, clockId, err := Parse("222236tg2qm22")
		if err != nil {
			t.Fatal(err)
		}

		if timestamp != 1234567890 {
			t.Fatal("invalid timestamp")
		}

		if clockId != 0 {
			t.Fatal("invalid clockId")
		}
	})
}
