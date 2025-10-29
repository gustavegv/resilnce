package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// ---- COOKIE PAYLOAD + HELPERS ----

type payload struct {
	UID string `json:"uid"`
	SID string `json:"sid"`
	Exp int64  `json:"exp"`
}

func FakeIncomingPayload() payload {
	const incomingUID = "user-123"
	const incomingSID = "sid-demo-abc"

	const expiryTime time.Duration = 120 * 24
	var incomingExp = time.Now().Add(expiryTime * time.Hour).Unix()

	pl := payload{
		UID: incomingUID, // demo value; normally from your DB
		SID: incomingSID, // 16 random bytes -> base64url
		Exp: incomingExp, // expires in 30 min (demo)
	}

	return pl
}

func MakeCookie(p payload, secret []byte) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sig := computeHMAC(raw, secret)
	raw64 := base64.RawURLEncoding.EncodeToString(raw)
	sig64 := base64.RawURLEncoding.EncodeToString(sig)
	cookie := raw64 + "." + sig64
	return cookie, nil
}

func VerifyCookie(cookie string, secret []byte) (payload, bool) {
	var out payload

	dotPos := -1
	for i := 0; i < len(cookie); i++ {
		if cookie[i] == '.' {
			dotPos = i
			break
		}
	}
	if dotPos <= 0 || dotPos >= len(cookie)-1 {
		return out, false
	}

	encPayload := cookie[:dotPos]
	encSig := cookie[dotPos+1:]

	rawPayload, err := base64.RawURLEncoding.DecodeString(encPayload)
	if err != nil {
		return out, false
	}
	rawSig, err := base64.RawURLEncoding.DecodeString(encSig)
	if err != nil {
		return out, false
	}

	expected := computeHMAC(rawPayload, secret)
	if !hmac.Equal(rawSig, expected) {
		return out, false
	}

	if err := json.Unmarshal(rawPayload, &out); err != nil {
		return out, false
	}

	if time.Now().Unix() > out.Exp {
		return out, false
	}

	return out, true
}

func computeHMAC(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
