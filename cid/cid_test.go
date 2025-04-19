package cid

import (
	"bytes"
	"testing"
)

func TestCreate(t *testing.T) {
	t.Run("cbor", func(t *testing.T) {
		c, err := Create(CodecCbor, []byte("abc"))
		if err != nil {
			t.Fatal(err)
		}

		if c.Version != Version {
			t.Fatal("invalid version")
		}

		if c.Codec != CodecCbor {
			t.Fatal("invalid codec")
		}

		if c.HashType != SHA256 {
			t.Fatal("invalid hash type")
		}

		if len(c.Digest) != 32 {
			t.Fatal("invalid digest length")
		}

		if len(c.Bytes) != 36 {
			t.Fatal("invalid bytes length")
		}

		if c.String() != "bafyreif2pall7dybz7vecqka3zo24irdwabwdi4wc55jznaq75q7eaavvu" {
			t.Fatal("invalid cid string")
		}
	})
}

func TestCreateEmpty(t *testing.T) {
	t.Run("cbor", func(t *testing.T) {
		c, err := CreateEmpty(CodecCbor)
		if err != nil {
			t.Fatal(err)
		}

		if c.Version != Version {
			t.Fatal("invalid version")
		}

		if c.Codec != CodecCbor {
			t.Fatal("invalid codec")
		}

		if c.HashType != SHA256 {
			t.Fatal("invalid hash type")
		}

		if len(c.Digest) != 0 {
			t.Fatal("invalid digest length")
		}

		if len(c.Bytes) != 4 {
			t.Fatal("invalid bytes length")
		}

		if c.String() != "bafyreaa" {
			t.Fatal("invalid cid string")
		}
	})
}

func TestParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		c, err := Parse("bafyreihffx5a2e7k5uwrmmgofbvzujc5cmw5h4espouwuxt3liqoflx3ee")
		if err != nil {
			t.Fatal(err)
		}

		if c.Version != Version {
			t.Fatal("invalid version")
		}

		if c.Codec != CodecCbor {
			t.Fatal("invalid codec")
		}

		if c.HashType != SHA256 {
			t.Fatal("invalid hash type")
		}

		testDigest := []byte{229, 45, 250, 13, 19, 234, 237, 45, 22, 48, 206, 40, 107, 154, 36, 93, 19, 45, 211, 240, 146, 123, 169, 106, 94, 123, 90, 32, 226, 174, 251, 33}
		if !bytes.Equal(c.Digest, testDigest) {
			t.Fatal("invalid digest")
		}

		testBytes := []byte{1, 113, 18, 32, 229, 45, 250, 13, 19, 234, 237, 45, 22, 48, 206, 40, 107, 154, 36, 93, 19, 45, 211, 240, 146, 123, 169, 106, 94, 123, 90, 32, 226, 174, 251, 33}
		if !bytes.Equal(c.Bytes, testBytes) {
			t.Fatal("invalid bytes")
		}
	})

	t.Run("empty", func(t *testing.T) {
		c, err := Parse("bafyreaa")
		if err != nil {
			t.Fatal(err)
		}

		if c.Version != Version {
			t.Fatal("invalid version")
		}

		if c.Codec != CodecCbor {
			t.Fatal("invalid codec")
		}

		if c.HashType != SHA256 {
			t.Fatal("invalid hash type")
		}

		if len(c.Digest) != 0 {
			t.Fatal("invalid digest length")
		}

		testBytes := []byte{1, 113, 18, 0}
		if !bytes.Equal(c.Bytes, testBytes) {
			t.Fatal("invalid bytes")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := Parse("QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
