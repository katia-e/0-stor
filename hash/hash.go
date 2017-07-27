package hash

import (
	"fmt"
)

// Hash Type
const (
	_ = iota
	TypeBlake2
	TypeSHA256
	TypeMD5
)

type hashEngine func([]byte) []byte

// Config defines hasher configuration
type Config struct {
	Type int
}

// Hasher is object that produces hash according to it's type
// given during it's creation
type Hasher struct {
	engine hashEngine
}

// NewHasher creates new hasher
func NewHasher(conf Config) (*Hasher, error) {
	var eng hashEngine

	switch conf.Type {
	case TypeBlake2:
		eng = blake2bHash
	case TypeSHA256:
		eng = sha256Hash
	case TypeMD5:
		eng = md5Hash
	default:
		return nil, fmt.Errorf("invalid hash type: %v", conf.Type)
	}
	return &Hasher{
		engine: eng,
	}, nil
}

// Hash produces hash of the given []byte
func (h Hasher) Hash(plain []byte) []byte {
	return h.engine(plain)
}
