package proxy

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/util"
)

const (
	magicConnect byte = iota
	magicClose
	magicData
	magicAck
	magicError
)

// TODO: use BSON!
type payloadConnect struct {
	Hops   []string `json:"hops"`
	Target struct {
		Port int    `json:"port"`
		IP   string `json:"ip"`
	} `json:"target"`
	Nonce     string `json:"nonce"`
	Signature []byte `json:"signature"`
}

func (c *payloadConnect) dice() {
	var (
		charset = "ABCDEF1234567890"
		idxs    = make([]byte, 32)
		nonce   = make([]byte, 32)
	)
	_, err := rand.Read(idxs)
	if err != nil {
		panic(err)
	}
	for i := range nonce {
		nonce[i] = charset[int(idxs[i])%len(charset)]
	}
	c.Nonce = string(nonce)
}

func (c *payloadConnect) diceAndSign(key []byte) {
	if len(key) <= 0 {
		return
	}
	c.dice()
	c.Signature = c.hash(key)
}

func (c *payloadConnect) hash(key []byte) []byte {
	concat := []byte(c.Nonce)
	concat = append(concat, '.')
	concat = append(concat, key...)
	checksum := sha1.Sum(concat)
	return checksum[:]
}

func (c *payloadConnect) valid(key []byte) bool {
	if len(c.Signature) <= 0 {
		return false
	}
	return util.EqualSlice(c.Signature, c.hash(key))
}

type payloadData struct {
	Data []byte `json:"data"`
}

type payloadError struct {
	Reason string `json:"reason"`
}

func encodeMsg(magic byte, payload any) []byte {
	if payload == nil {
		return []byte{magic}
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return append([]byte{magic}, buf...)
}
