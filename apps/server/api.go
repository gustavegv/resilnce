package main

import (
	"fmt"
	"net/http"
)

const secretKey int64 = 1234567890

func BaseReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "hello from cloud run: %s %s\n", r.Method, r.URL.Path)
}

func CookieReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	pl := FakeIncomingPayload()

	secret := []byte(fmt.Sprintf("%d", secretKey))
	cookie, _ := MakeCookie(pl, secret)

	fmt.Fprintf(w, "Cookie: %s\n", cookie)
	print("Cookie created\n")
}

func ValidateCookieReq(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	c, err := r.Cookie("sess")
	secret := []byte(fmt.Sprintf("%d", secretKey))
	println("Authenticating cookie...")

	if err == nil {
		fmt.Fprint(w, c.Value) // "CKI"
		_, b := VerifyCookie(c.Value, secret)
		if b {
			fmt.Fprintf(w, "Success, cookie real")
			println("Cookie real")

		} else {
			fmt.Fprintf(w, "Fail, cookie invalid")
			println("Cookie fake")

		}
	}
}
