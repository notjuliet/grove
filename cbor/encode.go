package cbor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/notjuliet/grove/cid"
)

type encState struct {
	b         []byte
	p         int // position
	currKey   *string
	currIndex *int
	currValue *any
}

func (s *encState) ensureWrite(needed int) {
	if s.p+needed <= len(s.b) || needed < 0 {
		return
	}

	currentLen := len(s.b)
	requiredLen := s.p + needed

	if currentLen < requiredLen {
		newLen := max(currentLen*2, requiredLen)
		newSlice := make([]byte, newLen)
		copy(newSlice, s.b)
		s.b = newSlice
	}
}

func (s *encState) writeUint8(val uint8) {
	s.ensureWrite(1)
	s.b[s.p] = val
	s.p++
}

func (s *encState) writeUint16(val uint16) {
	s.ensureWrite(2)
	binary.BigEndian.PutUint16(s.b[s.p:], val)
	s.p += 2
}

func (s *encState) writeUint32(val uint32) {
	s.ensureWrite(4)
	binary.BigEndian.PutUint32(s.b[s.p:], val)
	s.p += 4
}

func (s *encState) writeUint64(val uint64) {
	s.ensureWrite(8)
	binary.BigEndian.PutUint64(s.b[s.p:], val)
	s.p += 8
}

func (s *encState) writeFloat64(val float64) {
	s.writeUint8(0xe0 | 27)
	s.ensureWrite(8)
	binary.BigEndian.PutUint64(s.b[s.p:], math.Float64bits(val))
	s.p += 8
}

func (s *encState) writeTypeArgument(info byte, arg uint64) {
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
}

func (s *encState) writeBytes(val []byte, info byte) {
	s.writeTypeArgument(info, uint64(len(val)))
	s.ensureWrite(len(val))
	copy(s.b[s.p:s.p+len(val)], val)
	s.p += len(val)
}

func (s *encState) writeString(val string) {
	s.writeBytes([]byte(val), 3)
}

func (s *encState) writeCid(link cid.CidLink) {
	val := link.Bytes
	s.writeTypeArgument(6, 42)
	s.writeTypeArgument(2, uint64(len(val)+1))
	s.writeUint8(0x00)
	s.ensureWrite(len(val))
	copy(s.b[s.p:s.p+len(val)], val)
	s.p += len(val)
}

func (s *encState) writeAny(value any) error {
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

	case int, int8, int16, int32, int64:
		if v.(int64) >= 0 {
			s.writeTypeArgument(0, v.(uint64))
		} else {
			s.writeTypeArgument(1, uint64(-1-v.(int64)))
		}

	case uint, uint8, uint16, uint32, uint64:
		s.writeTypeArgument(0, v.(uint64))

	case float32, float64:
		s.writeFloat64(v.(float64))

	case []any:
		s.writeTypeArgument(4, uint64(len(v)))
		for i, elem := range v {
			if err := s.writeAny(elem); err != nil {
				s.currIndex = &i
				return err
			}
		}

	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}

		slices.SortFunc(keys, func(a, b string) int {
			lenA := len(a)
			lenB := len(b)
			if lenA != lenB {
				if lenA < lenB {
					return -1
				}
				return 1
			}
			return strings.Compare(a, b)
		})

		s.writeTypeArgument(5, uint64(len(v)))
		for _, key := range keys {
			s.writeString(key)
			if err := s.writeAny(v[key]); err != nil {
				s.currKey = &key
				return err
			}
		}

	case cid.CidLink:
		s.writeCid(v)

	default:
		s.currValue = &v
		return errors.New("Error while encoding CBOR")
	}

	return nil
}

func Encode(value map[string]any) ([]byte, error) {
	s := &encState{b: make([]byte, 1024)}

	if err := s.writeAny(value); err != nil {
		if s.currKey != nil {
			err = errors.Join(err, fmt.Errorf("failed encoding map value for key %s", *s.currKey))
		}
		if s.currIndex != nil {
			err = errors.Join(err, fmt.Errorf("failed encoding array element %d", *s.currIndex))
		}
		if s.currValue != nil {
			err = errors.Join(err, fmt.Errorf("unsupported type for CBOR encoding: %T", *s.currValue))
		}
		return nil, err
	}

	return s.b[:s.p], nil
}
