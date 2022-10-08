package b2b

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
)

type asymmetric struct {
	key *rsa.PrivateKey
	pub []byte
}

func GenerateAsymmetric() (*asymmetric, error) {

	a := &asymmetric{}

	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	a.pub = x509.MarshalPKCS1PublicKey(&k.PublicKey)
	a.key = k

	return a, nil
}

func (a *asymmetric) Encrypt(data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, &a.key.PublicKey, data)
}

func (a *asymmetric) Decrypt(data []byte) ([]byte, error) {
	return a.key.Decrypt(nil, data, nil)
}

func Encrypt(pub []byte, data []byte) ([]byte, error) {

	key, err := x509.ParsePKCS1PublicKey(pub)
	if err != nil {
		return nil, err
	}

	return rsa.EncryptPKCS1v15(rand.Reader, key, data)
}

type symmetric struct {
	a cipher.AEAD
}

func NewSymmetricKey(key []byte) (*symmetric, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &symmetric{a: aesGCM}, nil
}

func (s *symmetric) Encrypt(data []byte) ([]byte, error) {

	nonce := make([]byte, s.a.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	return s.a.Seal(nonce, nonce, data, nil), nil
}

func (s *symmetric) Decrypt(data []byte) ([]byte, error) {

	nonceSize := s.a.NonceSize()

	if len(data) < nonceSize {
		return nil, errors.New("insufficient data length")
	}

	//Extract the nonce from the encrypted data
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plaintext, err := s.a.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
