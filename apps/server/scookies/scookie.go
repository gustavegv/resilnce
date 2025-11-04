package scookie

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

type payload struct {
	UID  string `json:"uid"`
	SID  string `json:"sid"`
	Name string `json:"name"`
	Exp  int64  `json:"exp"`
}

func BuildPayload(uID, SID, name string) payload {
	var pl payload
	pl.UID = uID
	pl.SID = SID
	pl.Name = name

	const expiryTime time.Duration = 120 * 24
	var incomingExp = time.Now().Add(expiryTime * time.Hour).Unix()
	pl.Exp = incomingExp

	return pl
}

func Create(p payload, secret []byte) (string, error) {
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

func Verify(cookie string, secret []byte) (payload, bool) {
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

func GetSecret() []byte {
	sec64 := os.Getenv("SIGNED_COOKIE_SECRET")
	if sec64 == "" {
		log.Fatal("Signed cookie secret not found")
	}
	secret, err := base64.RawURLEncoding.DecodeString(sec64)

	if err != nil {
		log.Fatalf("invalid base64 secret: %v", err)
	}
	if len(secret) < 32 {
		log.Fatal("secret too short; need >=32 bytes. current length:", len(secret))
	}
	return secret
}

func CreateNewSecret(nChars int) (string, error) {
	if nChars <= 0 {
		return "", nil
	}
	bytesNeeded := (nChars*3 + 3) / 4

	var sb strings.Builder
	for sb.Len() < nChars {
		b := make([]byte, bytesNeeded)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		sb.WriteString(base64.RawURLEncoding.EncodeToString(b))
	}
	s := sb.String()
	return s[:nChars], nil
}
