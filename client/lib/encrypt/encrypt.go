package encrypt

import (
	"errors"
	"fmt"

	"github.com/zero-os/0-stor/client/lib/block"
)

// Encryption type
const (
	TypeAESGCM = "aes_gcm"
)

var (
	errInvalidPrivKeyLen = errors.New("invalid private key length")
	errInvalidNonceLen   = errors.New("invalid nonce length")
)

// Config defines EncrypterDecrypter config
type Config struct {
	Type    string `yaml:"type"`
	PrivKey string `yaml:"privKey"`
	Nonce   string `yaml:"nonce"`
}

// EncrypterDecrypter is interaface for encrypter and decrypter
type EncrypterDecrypter interface {
	Encrypt(plain []byte) []byte
	Decrypt(cipher []byte) (plain []byte, err error)
}

// NewEncrypterDecrypter creates new EncrypterDecrypter
func NewEncrypterDecrypter(conf Config) (EncrypterDecrypter, error) {
	switch conf.Type {
	case TypeAESGCM:
		return newAESGCM([]byte(conf.PrivKey), []byte(conf.Nonce))
	default:
		return nil, fmt.Errorf("invalid type: %v", conf.Type)
	}
}

// Writer defines encryption writer
type Writer struct {
	ed EncrypterDecrypter
	w  block.Writer
}

// NewWriter creates new encryption writer
func NewWriter(w block.Writer, conf Config) (*Writer, error) {
	ed, err := NewEncrypterDecrypter(conf)
	if err != nil {
		return nil, err
	}
	return &Writer{
		w:  w,
		ed: ed,
	}, nil
}

// WriteBlock implements blockreadwrite.Writer interface
func (w Writer) WriteBlock(key, plain []byte) (int, error) {
	encrypted := w.ed.Encrypt(plain)
	return w.w.WriteBlock(key, encrypted)
}

// Reader defines encryption reader.
// It use ioutil.ReadAll so it won't save your memory usage
type Reader struct {
	ed EncrypterDecrypter
}

// NewReader creates new encryption reader
func NewReader(conf Config) (*Reader, error) {
	ed, err := NewEncrypterDecrypter(conf)
	if err != nil {
		return nil, err
	}
	return &Reader{
		ed: ed,
	}, nil
}

// ReadBlock implements block.Reader.
func (r *Reader) ReadBlock(data []byte) ([]byte, error) {
	return r.ed.Decrypt(data)
}
