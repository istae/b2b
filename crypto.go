package b2b

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"errors"
)

type asymmetric struct {
	priv     *rsa.PrivateKey
	pub      *rsa.PublicKey
	pubBytes []byte
}

func NewAsymmetric() (*asymmetric, error) {

	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return &asymmetric{
		priv:     k,
		pub:      &k.PublicKey,
		pubBytes: x509.MarshalPKCS1PublicKey(&k.PublicKey),
	}, nil
}

func (a *asymmetric) Encrypt(data []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, a.pub, data)
}

func (a *asymmetric) Decrypt(data []byte) ([]byte, error) {
	return a.priv.Decrypt(nil, data, nil)
}

func (a *asymmetric) Sign(data []byte) ([]byte, error) {
	h := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, a.priv, crypto.SHA256, h[:])
}

func (a *asymmetric) Verify(data, sig []byte) error {
	h := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(a.pub, crypto.SHA256, h[:], sig)
}

func (a *asymmetric) Unmarshal(pub, priv []byte) error {

	if pub != nil {
		key, err := x509.ParsePKCS1PublicKey(pub)
		if err != nil {
			return err
		}
		a.pub = key
		a.pubBytes = pub
	}

	if priv != nil {
		key, err := x509.ParsePKCS1PrivateKey(priv)
		if err != nil {
			return err
		}
		a.priv = key
	}

	return nil
}

func (a *asymmetric) Marshal() ([]byte, []byte) {
	return x509.MarshalPKCS1PublicKey(a.pub), x509.MarshalPKCS1PrivateKey(a.priv)
}

func Hash(b []byte) []byte {
	a := sha256.Sum256(b)
	return a[:]
}

type symmetric struct {
	a cipher.AEAD
}

func RandomKey() []byte {
	id := make([]byte, LengthKey)
	_, _ = rand.Read(id)
	return id
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
