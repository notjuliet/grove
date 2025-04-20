package cbor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
)

type CidLink struct {
	Bytes []byte
}

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
	return val, nil
}

func (s *state) readArgument(info byte) (uint64, error) {
	if info < 24 {
		return uint64(info), nil
	}

	switch info {
	case 24:
		val, err := s.readUint8()
		return uint64(val), err
	case 25:
		val, err := s.readUint16()
		return uint64(val), err
	case 26:
		val, err := s.readUint32()
		return uint64(val), err
	case 27:
		val, err := s.readUint64()
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
	return string(bytes), nil
}

func (s *state) readTypeInfo() (majorType byte, info byte, err error) {
	prelude, err := s.readUint8()
	if err != nil {
		return 0, 0, err
	}
	return prelude >> 5, prelude & 0x1f, nil
}

func (s *state) readCid(length uint64) (CidLink, error) {
	if length == 0 {
		return CidLink{}, fmt.Errorf("invalid CID encoding: length %d too short for prefix", length)
	}

	if err := s.ensureRead(int(length)); err != nil {
		return CidLink{}, fmt.Errorf("reading CID: %w", err)
	}

	prefix := s.b[s.p]
	if prefix != 0x00 {
		return CidLink{}, fmt.Errorf("invalid CID encoding: expected 0x00 prefix, got 0x%02x", prefix)
	}

	cidLen := int(length - 1)
	cidBytes := make([]byte, cidLen)
	copy(cidBytes, s.b[s.p+1:s.p+int(length)])
	s.p += int(length)

	return CidLink{Bytes: cidBytes}, nil
}

type container struct {
	isMap     bool       // true for map, false for array
	elements  any        // *[]any or *map[string]any
	mapKey    *string    // Holds the current key while decoding map value
	remaining uint64     // Number of items (or key/value pairs * 2 for maps) left
	next      *container // Link to parent container
}

func DecodeFirst(buf []byte) (value map[string]any, remainder []byte, err error) {
	if len(buf) == 0 {
		return nil, buf, errors.New("input buffer is empty")
	}

	s := &state{b: buf, p: 0}
	var stack *container = nil
	var currVal any

	for s.p < len(s.b) {
		majorType, info, err := s.readTypeInfo()
		if err != nil {
			return nil, buf, fmt.Errorf("reading type info: %w", err)
		}

		var arg uint64
		if majorType < 7 {
			arg, err = s.readArgument(info)
			if err != nil {
				return nil, buf, fmt.Errorf("reading argument for type %d: %w", majorType, err)
			}
		}

		switch majorType {
		case 0: // Unsigned Integer
			currVal = arg
		case 1: // Negative Integer
			if arg > math.MaxInt64 {
				return nil, buf, fmt.Errorf("negative integer -1-%d is too large (overflows int64)", arg)
			}
			currVal = -1 - int64(arg)
		case 2: // Byte String
			currVal, err = s.readBytes(arg)
			if err != nil {
				return nil, buf, err
			}
		case 3: // Text String
			currVal, err = s.readString(arg)
			if err != nil {
				return nil, buf, err
			}
		case 4: // Array
			arr := make([]any, 0, int(arg))
			currVal = &arr
			if arg > 0 {
				stack = &container{
					isMap:     false,
					elements:  currVal,
					remaining: arg,
					next:      stack,
				}
				continue
			}
		case 5: // Map
			m := make(map[string]any, int(arg))
			currVal = &m
			if arg > 0 {
				stack = &container{
					isMap:     true,
					elements:  currVal,
					remaining: arg * 2,
					mapKey:    nil,
					next:      stack,
				}
				continue
			}
		case 6: // Tag
			switch arg {
			case 42: // CID Link
				contentMajorType, contentInfo, err := s.readTypeInfo()
				if err != nil {
					return nil, buf, fmt.Errorf("reading type info for tag %d content: %w", arg, err)
				}
				if contentMajorType != 2 {
					return nil, buf, fmt.Errorf("expected tag %d content to be type 2 (bytes), got type %d", arg, contentMajorType)
				}
				contentArg, err := s.readArgument(contentInfo)
				if err != nil {
					return nil, buf, fmt.Errorf("reading argument for tag %d content: %w", arg, err)
				}
				currVal, err = s.readCid(contentArg)
				if err != nil {
					return nil, buf, fmt.Errorf("reading CID for tag %d: %w", arg, err)
				}
			default:
				return nil, buf, fmt.Errorf("unsupported tag number: %d", arg)
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
					return nil, buf, err
				}
			default:
				return nil, buf, fmt.Errorf("invalid simple value info: %d", info)
			}
		default:
			return nil, buf, fmt.Errorf("internal error: invalid major type %d", majorType)
		}

		for stack != nil {
			if stack.isMap {
				mapPtr := stack.elements.(*map[string]any)
				if stack.mapKey == nil {
					keyStr, ok := currVal.(string)
					if !ok {
						return nil, buf, fmt.Errorf("map key must be a string, got %T", currVal)
					}
					stack.mapKey = &keyStr
				} else {
					(*mapPtr)[*stack.mapKey] = currVal
					stack.mapKey = nil
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

	return currVal.(map[string]any), s.b[s.p:], nil
}

func Decode(buf []byte) (map[string]any, error) {
	val, rmd, err := DecodeFirst(buf)
	if err != nil {
		return nil, err
	}
	if len(rmd) != 0 {
		return val, fmt.Errorf("decoding finished with %d remaining bytes", len(rmd))
	}
	return val, nil
}
