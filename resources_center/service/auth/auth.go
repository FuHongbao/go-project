package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"hash"
	"io"
	"strings"
)

func GetHeaderSignature(method, AccSecret string, md5Value, contentType, Date string, ossHeaders, Resource string) (signedStr string, err error) {
	if Date == "" || Resource == "" {
		err = errors.New("signature failed Data or CanonicalizedResource is nil")
		return
	}
	var signatureStr = []string{method, "\n", md5Value, "\n", contentType, "\n", Date, "\n", ossHeaders, Resource}
	signature := strings.Join(signatureStr, "")
	h := hmac.New(func() hash.Hash { return sha1.New() }, []byte(AccSecret))
	_, err = io.WriteString(h, signature)
	if err != nil {
		return
	}
	signedStr = base64.StdEncoding.EncodeToString(h.Sum(nil))
	return
}

func GetAuthorization(AccKey, signature string) string {
	return "OSS " + AccKey + ":" + signature
}
