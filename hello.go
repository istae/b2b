package b2b

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"
)

func (b *b2b) sayHello(c net.Conn) (peerID string, symmetricKey *symmetric, err error) {

	n := time.Now()
	defer func(t time.Time) {
		fmt.Println("handshake took", time.Since(t).Milliseconds())
	}(n)

	var (
		insecure = insecureReadWriter{conn: c}
		peerPub  = &asymmetric{}
	)

	var (
		msg = &Msg{
			Protocol: helloProcol,
			StreamID: randomID(),
			PeerID:   b.peerID,
			Data:     b.key.pubBytes,
		}
	)

	// send public key
	err = insecure.Write(msg)
	if err != nil {
		return
	}

	// get public key
	err = insecure.Read(msg)
	if err != nil {
		return
	}
	peerID = msg.PeerID
	pubKey := msg.Data

	// verify peerID
	if peerID != hex.EncodeToString(Hash(pubKey)) {
		err = errPeerIDVerification
		return
	}
	err = peerPub.Unmarshal(pubKey, nil)
	if err != nil {
		return
	}

	// send key
	key := RandomKey()
	msg.PeerID = b.peerID
	msg.Data = key
	err = NewSecureReadWriter(c, peerPub, b.key).Write(msg)
	if err != nil {
		return
	}

	// send sig
	msg.Data, err = b.key.Sign(key)
	if err != nil {
		return
	}
	err = insecure.Write(msg)
	if err != nil {
		return
	}

	// get sig and verify
	err = insecure.Read(msg)
	if err != nil {
		return
	}
	err = peerPub.Verify(key, msg.Data)
	if err != nil {
		return
	}

	symmetricKey, err = NewSymmetric(key)

	return
}

func (b *b2b) sayHelloBack(c net.Conn) (peerID string, symmetricKey *symmetric, err error) {

	var (
		insecure = insecureReadWriter{conn: c}
		peerPub  = &asymmetric{}
		msg      = &Msg{}
	)

	// get public key
	err = insecure.Read(msg)
	if err != nil {
		return
	}
	peerID = msg.PeerID
	pubKey := msg.Data

	// verify peerID
	if peerID != hex.EncodeToString(Hash(pubKey)) {
		err = errPeerIDVerification
		return
	}
	err = peerPub.Unmarshal(msg.Data, nil)
	if err != nil {
		return
	}

	// send public key
	msg.PeerID = b.peerID
	msg.Data = b.key.pubBytes
	err = insecure.Write(msg)
	if err != nil {
		return
	}

	// get key
	err = NewSecureReadWriter(c, peerPub, b.key).Read(msg)
	if err != nil {
		return
	}
	key := msg.Data

	// get sig and verify
	err = insecure.Read(msg)
	if err != nil {
		return
	}
	err = peerPub.Verify(key, msg.Data)
	if err != nil {
		return
	}

	// send sig
	msg.PeerID = b.peerID
	msg.Data, err = b.key.Sign(key)
	if err != nil {
		return
	}
	err = insecure.Write(msg)
	if err != nil {
		return
	}

	symmetricKey, err = NewSymmetric(key)

	return
}
