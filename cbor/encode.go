package cbor

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/notjuliet/grove/cid"
)

type State struct {
	b []byte
	p int // position
}

func (s *State) ensureCapacity(needed int) error {
	if needed < 0 {
		return fmt.Errorf("needed count cannot be negative")
	}

	currentLen := len(s.b)
	currentCap := cap(s.b)
	requiredCap := currentLen + needed

	if currentCap < requiredCap {
		newCapacity := max(currentCap*2, requiredCap)

		newSlice := make([]byte, newCapacity)
		copy(newSlice, s.b)
		s.b = newSlice
	}

	return nil
}

func (s *State) ensureWrite(n int) error {
	if s.p+n > len(s.b) {
		if err := s.ensureCapacity(n); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) writeUint8(val uint8) error {
	if err := s.ensureWrite(1); err != nil {
		return err
	}
	s.b[s.p] = val
	s.p++
	return nil
}

func (s *State) writeUint16(val uint16) error {
	if err := s.ensureWrite(2); err != nil {
		return err
	}
	binary.BigEndian.PutUint16(s.b[s.p:], val)
	s.p += 2
	return nil
}

func (s *State) writeUint32(val uint32) error {
	if err := s.ensureWrite(4); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(s.b[s.p:], val)
	s.p += 4
	return nil
}

func (s *State) writeUint64(val uint64) error {
	if err := s.ensureWrite(8); err != nil {
		return err
	}
	binary.BigEndian.PutUint64(s.b[s.p:], val)
	s.p += 8
	return nil
}

func (s *State) writeFloat64(val float64) error {
	if err := s.ensureWrite(8); err != nil {
		return err
	}
	s.writeUint8(0xe0 | 27)
	binary.BigEndian.PutUint64(s.b[s.p:], math.Float64bits(val))
	s.p += 8
	return nil
}

func (s *State) writeTypeArgument(info byte, arg uint64) error {
	if arg < 24 {
		s.writeUint8(info<<5 | byte(arg))
	} else if arg < 0x100 {
		s.writeUint8(info<<5 | 24)
		s.writeUint8(uint8(arg))
	} else if arg < 0x10000 {
		s.writeUint8(info<<5 | 25)
		s.writeUint16(uint16(arg))
	} else if arg < 0x100000000 {
		s.writeUint8(info<<5 | 26)
		s.writeUint32(uint32(arg))
	} else {
		s.writeUint8(info<<5 | 27)
		s.writeUint64(arg)
	}

	return nil
}

func (s *State) writeBytes(val []byte, info byte) error {
	if err := s.writeTypeArgument(info, uint64(len(val))); err != nil {
		return err
	}
	if err := s.ensureWrite(len(val)); err != nil {
		return err
	}
	copy(s.b[s.p:s.p+len(val)], val)
	s.p += len(val)
	return nil
}

func (s *State) writeString(val string) error {
	return s.writeBytes([]byte(val), 3)
}

func (s *State) writeCid(link cid.CidLink) error {
	val := link.Bytes
	if err := s.writeTypeArgument(6, 42); err != nil {
		return err
	}
	if err := s.writeTypeArgument(2, uint64(len(val)+1)); err != nil {
		return err
	}
	if err := s.writeUint8(0x00); err != nil {
		return err
	}
	if err := s.ensureWrite(len(val)); err != nil {
		return err
	}
	copy(s.b[s.p:s.p+len(val)], val)
	s.p += len(val)

	return nil
}

func (s *State) writeAny(value any) error {
	switch v := value.(type) {
	case nil:
		s.writeUint8(0xf6)
	case bool:
		if v {
			s.writeUint8(0xf5)
		} else {
			s.writeUint8(0xf4)
		}

	case string:
		s.writeString(v)
	case []byte:
		s.writeBytes(v, 2)

	case int:
		if v >= 0 {
			s.writeTypeArgument(0, uint64(v))
		} else {
			s.writeTypeArgument(1, uint64(-1-v))
		}
	case int8:
		if v >= 0 {
			s.writeTypeArgument(0, uint64(v))
		} else {
			s.writeTypeArgument(1, uint64(-1-v))
		}
	case int16:
		if v >= 0 {
			s.writeTypeArgument(0, uint64(v))
		} else {
			s.writeTypeArgument(1, uint64(-1-v))
		}
	case int32:
		if v >= 0 {
			s.writeTypeArgument(0, uint64(v))
		} else {
			s.writeTypeArgument(1, uint64(-1-v))
		}
	case int64:
		if v >= 0 {
			s.writeTypeArgument(0, uint64(v))
		} else {
			s.writeTypeArgument(1, uint64(-1-v))
		}

	case uint:
		s.writeTypeArgument(0, uint64(v))
	case uint8:
		s.writeTypeArgument(0, uint64(v))
	case uint16:
		s.writeTypeArgument(0, uint64(v))
	case uint32:
		s.writeTypeArgument(0, uint64(v))
	case uint64:
		s.writeTypeArgument(0, uint64(v))

	case float32:
		s.writeFloat64(float64(v))
	case float64:
		s.writeFloat64(v)

	case []any:
		s.writeTypeArgument(4, uint64(len(v)))
		for i, elem := range v {
			if err := s.writeAny(elem); err != nil {
				return fmt.Errorf("failed encoding array element %d: %w", i, err)
			}
		}

	case map[string]any:
		s.writeTypeArgument(5, uint64(len(v)))
		for key, val := range v {
			if err := s.writeString(key); err != nil {
				return fmt.Errorf("failed encoding map key %s: %w", key, err)
			}
			if err := s.writeAny(val); err != nil {
				return fmt.Errorf("failed encoding map value for key %s: %w", key, err)
			}
		}

	case cid.CidLink:
		if err := s.writeCid(v); err != nil {
			return fmt.Errorf("failed encoding cid-link: %w", err)
		}

	default:
		return fmt.Errorf("unsupported type for CBOR encoding: %T", v)
	}

	return nil
}

func Encode(value map[string]any) ([]byte, error) {
	s := &State{b: make([]byte, 1024)}

	if err := s.writeAny(value); err != nil {
		return nil, err
	}

	return s.b[:s.p], nil
}
