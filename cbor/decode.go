package cbor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"

	"github.com/notjuliet/grove/cid"
)

type state struct {
	b []byte
	p int // position
}

func (s *state) ensureRead(n int) error {
	if s.p+n > len(s.b) {
		return fmt.Errorf("unexpected end of input: need %d bytes, have %d", n, len(s.b)-s.p)
	}
	return nil
}

func (s *state) readUint8() (byte, error) {
	if err := s.ensureRead(1); err != nil {
		return 0, err
	}
	val := s.b[s.p]
	s.p++
	return val, nil
}

func (s *state) readUint16() (uint16, error) {
	if err := s.ensureRead(2); err != nil {
		return 0, err
	}
	val := binary.BigEndian.Uint16(s.b[s.p:])
	s.p += 2
	return val, nil
}

func (s *state) readUint32() (uint32, error) {
	if err := s.ensureRead(4); err != nil {
		return 0, err
	}
	val := binary.BigEndian.Uint32(s.b[s.p:])
	s.p += 4
	return val, nil
}

func (s *state) readUint64() (uint64, error) {
	if err := s.ensureRead(8); err != nil {
		return 0, err
	}
	val := binary.BigEndian.Uint64(s.b[s.p:])
	s.p += 8
	return val, nil
}

func (s *state) readFloat64() (float64, error) {
	if err := s.ensureRead(8); err != nil {
		return 0, err
	}
	val := math.Float64frombits(binary.BigEndian.Uint64(s.b[s.p:]))
	s.p += 8
	if math.IsNaN(val) {
		return 0, fmt.Errorf("decoded float is NaN, which is not allowed")
	}
	if math.IsInf(val, 0) {
		return 0, fmt.Errorf("decoded float is infinite, which is not allowed")
	}
	return val, nil
}

func (s *state) readArgument(info byte) (uint64, error) {
	if info < 24 {
		return uint64(info), nil
	}

	switch info {
	case 24:
		val, err := s.readUint8()
		if val < 24 {
			return 0, fmt.Errorf("integer is not minimally encoded")
		}
		return uint64(val), err
	case 25:
		val, err := s.readUint16()
		if val < math.MaxUint8 {
			return 0, fmt.Errorf("integer is not minimally encoded")
		}
		return uint64(val), err
	case 26:
		val, err := s.readUint32()
		if val < math.MaxUint16 {
			return 0, fmt.Errorf("integer is not minimally encoded")
		}
		return uint64(val), err
	case 27:
		val, err := s.readUint64()
		if val < math.MaxUint32 {
			return 0, fmt.Errorf("integer is not minimally encoded")
		}
		return val, err
	default:
		return 0, fmt.Errorf("invalid argument encoding info: %d", info)
	}
}

func (s *state) readBytes(length uint64) ([]byte, error) {
	if length > uint64(len(s.b)-s.p) {
		return nil, fmt.Errorf("unexpected end of input reading bytes: need %d, have %d", length, len(s.b)-s.p)
	}
	slice := make([]byte, length)
	copy(slice, s.b[s.p:s.p+int(length)])
	s.p += int(length)
	return slice, nil
}

func (s *state) readString(length uint64) (string, error) {
	bytes, err := s.readBytes(length)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(bytes) {
		return "", fmt.Errorf("invalid UTF-8 string")
	}
	return string(bytes), nil
}

func (s *state) readTypeInfo() (majorType byte, info byte, err error) {
	prelude, err := s.readUint8()
	if err != nil {
		return 0, 0, err
	}
	return prelude >> 5, prelude & 0x1f, nil
}

func (s *state) readCid(length uint64) (cid.CidLink, error) {
	if length == 0 {
		return cid.CidLink{}, fmt.Errorf("invalid CID encoding: length %d too short for prefix", length)
	}

	if err := s.ensureRead(int(length)); err != nil {
		return cid.CidLink{}, fmt.Errorf("reading CID: %w", err)
	}

	prefix := s.b[s.p]
	if prefix != 0x00 {
		return cid.CidLink{}, fmt.Errorf("invalid CID encoding: expected 0x00 prefix, got 0x%02x", prefix)
	}

	cidLen := int(length - 1)
	if cidLen <= 0 {
		return cid.CidLink{}, fmt.Errorf("invalid CID length")
	}
	cidBytes := make([]byte, cidLen)
	copy(cidBytes, s.b[s.p+1:s.p+int(length)])
	c := cid.CidLink{Bytes: cidBytes}
	if _, err := cid.Parse(c.String()); err != nil {
		return cid.CidLink{}, fmt.Errorf("invalid CID: %w", err)
	}
	s.p += int(length)
	return c, nil
}

type container struct {
	isMap           bool       // true for map, false for array
	elements        any        // *[]any or *map[string]any
	currMapKey      *string    // Holds the current key while decoding map value
	prevMapKeyBytes []byte     // Stores the raw bytes of the previous map key for DAG-CBOR sorting comparison
	remaining       uint64     // Number of items (or key/value pairs * 2 for maps) left
	next            *container // Link to parent container
}

func DecodeFirst(buf []byte) (value any, remainder []byte, err error) {
	if len(buf) == 0 {
		return nil, nil, errors.New("input buffer is empty")
	}

	s := &state{b: buf, p: 0}
	var stack *container = nil
	var currVal any

	for s.p < len(s.b) {
		majorType, info, err := s.readTypeInfo()
		if err != nil {
			return nil, s.b[s.p:], fmt.Errorf("reading type info: %w", err)
		}

		var arg uint64
		if majorType < 7 {
			arg, err = s.readArgument(info)
			if err != nil {
				return nil, s.b[s.p:], fmt.Errorf("reading argument for type %d: %w", majorType, err)
			}
		}

		switch majorType {
		case 0: // Unsigned Integer
			currVal = arg
		case 1: // Negative Integer
			currVal = -1 - int64(arg)
		case 2: // Byte String
			currVal, err = s.readBytes(arg)
			if err != nil {
				return nil, s.b[s.p:], err
			}
		case 3: // Text String
			currVal, err = s.readString(arg)
			if err != nil {
				return nil, s.b[s.p:], err
			}
		case 4: // Array
			arr := make([]any, 0, int(arg))
			if arg > 0 {
				currVal = &arr
				stack = &container{
					isMap:     false,
					elements:  currVal,
					remaining: arg,
					next:      stack,
				}
				continue
			}
			currVal = arr
		case 5: // Map
			m := make(map[string]any, int(arg))
			if arg > 0 {
				currVal = &m
				stack = &container{
					isMap:      true,
					elements:   currVal,
					remaining:  arg * 2,
					currMapKey: nil,
					next:       stack,
				}
				continue
			}
			currVal = m
		case 6: // Tag
			switch arg {
			case 42: // CID Link
				contentMajorType, contentInfo, err := s.readTypeInfo()
				if err != nil {
					return nil, s.b[s.p:], fmt.Errorf("reading type info for tag %d content: %w", arg, err)
				}
				if contentMajorType != 2 {
					return nil, s.b[s.p:], fmt.Errorf("expected tag %d content to be type 2 (bytes), got type %d", arg, contentMajorType)
				}
				contentArg, err := s.readArgument(contentInfo)
				if err != nil {
					return nil, s.b[s.p:], fmt.Errorf("reading argument for tag %d content: %w", arg, err)
				}
				currVal, err = s.readCid(contentArg)
				if err != nil {
					return nil, s.b[s.p:], fmt.Errorf("reading CID for tag %d: %w", arg, err)
				}
			default:
				return nil, s.b[s.p:], fmt.Errorf("unsupported tag number: %d", arg)
			}
		case 7: // Simple values and floats
			switch info {
			case 20: // False
				currVal = false
			case 21: // True
				currVal = true
			case 22: // Null
				currVal = nil
			case 27: // Float64
				currVal, err = s.readFloat64()
				if err != nil {
					return nil, s.b[s.p:], err
				}
			default:
				return nil, s.b[s.p:], fmt.Errorf("invalid simple value info: %d", info)
			}
		default:
			return nil, s.b[s.p:], fmt.Errorf("internal error: invalid major type %d", majorType)
		}

		for stack != nil {
			if stack.isMap {
				mapPtr := stack.elements.(*map[string]any)
				if stack.currMapKey == nil {
					keyStr, ok := currVal.(string)
					if !ok {
						return nil, s.b[s.p:], fmt.Errorf("map key must be a string, got %T (value: %v)", currVal, currVal)
					}
					currentKeyBytes := []byte(keyStr)

					// DAG-CBOR key ordering check
					if stack.prevMapKeyBytes != nil {
						if len(currentKeyBytes) < len(stack.prevMapKeyBytes) {
							return nil, s.b[s.p:], fmt.Errorf("map key order violation: key '%s' (len %d) is shorter than previous key '%s' (len %d)",
								keyStr, len(currentKeyBytes), string(stack.prevMapKeyBytes), len(stack.prevMapKeyBytes))
						}

						if len(currentKeyBytes) == len(stack.prevMapKeyBytes) {
							comparison := bytes.Compare(currentKeyBytes, stack.prevMapKeyBytes)
							if comparison == 0 {
								return nil, s.b[s.p:], fmt.Errorf("map key order violation: duplicate key '%s'", keyStr)
							}
							if comparison < 0 {
								return nil, s.b[s.p:], fmt.Errorf("map key order violation: key '%s' is lexicographically smaller than previous key '%s' of the same length",
									keyStr, string(stack.prevMapKeyBytes))
							}
						}
					}
					stack.prevMapKeyBytes = currentKeyBytes
					stack.currMapKey = &keyStr
				} else {
					(*mapPtr)[*stack.currMapKey] = currVal
					stack.currMapKey = nil
				}
			} else {
				arrPtr := stack.elements.(*[]any)
				*arrPtr = append(*arrPtr, currVal)
			}

			stack.remaining--
			if stack.remaining == 0 {
				currVal = reflect.ValueOf(stack.elements).Elem().Interface()
				stack = stack.next
			} else {
				goto nextItem
			}
		}
		break
	nextItem:
	}

	return currVal, s.b[s.p:], nil
}

func Decode(buf []byte) (any, error) {
	val, rmd, err := DecodeFirst(buf)
	if err != nil {
		return nil, err
	}
	if len(rmd) != 0 {
		return val, fmt.Errorf("decoding finished with %d remaining bytes", len(rmd))
	}
	return val, nil
}
