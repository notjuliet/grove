package cid

import (
	"crypto/sha256"
	"encoding/base32"
	"errors"
)

const (
	Version   = 1
	SHA256    = 0x12
	CodecRaw  = 0x55
	CodecCbor = 0x71
)

var b32Encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

// CID represents a Content Identifier.
//
// https://dasl.ing/cid.html
type Cid struct {
	// CID version, always 1 for CIDv1.
	Version int
	// Multicodec type, either 0x55 (raw) or 0x71 (DAG-CBOR).
	Codec int
	// Multicodec digest type, only 0x12 (SHA-256) is supported.
	HashType int
	// Raw digest value.
	Digest []byte
	// Raw CID bytes.
	Bytes []byte
}

func Create(codec int, value []byte) (Cid, error) {
	if codec != CodecRaw && codec != CodecCbor {
		return Cid{}, errors.New("invalid codec")
	}

	digest := sha256.Sum256(value)
	if len(digest) != 32 {
		return Cid{}, errors.New("invalid digest length")
	}

	// a SHA-256 CIDv1 is 36 bytes long, 4 bytes for the header, 32 bytes for the digest.
	bytes := make([]byte, 36)
	bytes[0] = Version
	bytes[1] = byte(codec)
	bytes[2] = SHA256
	bytes[3] = 32

	copy(bytes[4:], digest[:])

	return Cid{Version, codec, SHA256, digest[:], bytes}, nil
}

func CreateEmpty(codec int) (Cid, error) {
	if codec != CodecRaw && codec != CodecCbor {
		return Cid{}, errors.New("invalid codec")
	}

	bytes := make([]byte, 4)
	bytes[0] = Version
	bytes[1] = byte(codec)
	bytes[2] = SHA256
	bytes[3] = 0

	return Cid{Version, codec, SHA256, nil, bytes}, nil
}

func decode(bytes []byte) (Cid, error) {
	length := len(bytes)

	if length < 4 {
		return Cid{}, errors.New("cid too short")
	}

	version := bytes[0]
	codec := bytes[1]
	hashType := bytes[2]
	digestSize := bytes[3]

	if version != Version {
		return Cid{}, errors.New("invalid version")
	}

	if codec != CodecRaw && codec != CodecCbor {
		return Cid{}, errors.New("invalid codec")
	}

	if hashType != SHA256 {
		return Cid{}, errors.New("invalid hash type")
	}

	if digestSize != 32 && digestSize != 0 {
		return Cid{}, errors.New("invalid digest size")
	}

	if length < 4+int(digestSize) {
		return Cid{}, errors.New("cid too short")
	}

	digest := bytes[4 : 4+digestSize]
	remainder := bytes[4+digestSize:]

	if len(remainder) != 0 {
		return Cid{}, errors.New("cid bytes includes remainder")
	}

	return Cid{Version, int(codec), int(hashType), digest, bytes[0 : 4+digestSize]}, nil
}

func Parse(s string) (Cid, error) {
	if len(s) < 2 || s[0] != 'b' {
		return Cid{}, errors.New("invalid cid format")
	}

	// 4 bytes in base32 = 8 characters
	// 36 bytes in base32 = 59 characters
	if len(s) != 59 && len(s) != 8 {
		return Cid{}, errors.New("invalid cid length")
	}

	bytes, err := b32Encoding.DecodeString(s[1:])
	if err != nil {
		return Cid{}, err
	}

	cid, err := decode(bytes)
	if err != nil {
		return Cid{}, err
	}

	return cid, nil
}

func (c Cid) String() string {
	return "b" + b32Encoding.EncodeToString(c.Bytes)
}

func FromBytes(bytes []byte) (Cid, error) {
	// 4 bytes + 1 byte for the 0x00 prefix
	// 36 bytes + 1 byte for the 0x00 prefix
	if len(bytes) != 37 && len(bytes) != 5 {
		return Cid{}, errors.New("invalid cid length")
	}

	if bytes[0] != 0 {
		return Cid{}, errors.New("incorrect binary cid")
	}

	return decode(bytes[1:])
}

func (c Cid) ToBytes() []byte {
	return c.Bytes
}
