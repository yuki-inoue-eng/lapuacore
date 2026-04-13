package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

type signer struct {
	credential gateways.Credential
}

func newSigner(credential gateways.Credential) *signer {
	return &signer{
		credential: credential,
	}
}

func (s *signer) signHttpGet(req *http.Request) {
	s.signHttp(req, time.Now().UnixMilli(), req.URL.RawQuery)
}

func (s *signer) signHttpPost(req *http.Request, body []byte) {
	s.signHttp(req, time.Now().UnixMilli(), string(body[:]))
}

func (s *signer) signHttp(req *http.Request, timestamp int64, data string) {
	apiKey := s.credential.GetApiKey()
	secret := s.credential.GetSecret()
	timestampStr := strconv.FormatInt(timestamp, 10)
	hmac256 := hmac.New(sha256.New, []byte(secret))
	hmac256.Write([]byte(timestampStr + apiKey + data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestampStr)
	req.Header.Set("X-BAPI-SIGN", hex.EncodeToString(hmac256.Sum(nil)))
	req.Header.Set("X-BAPI-SIGN-TYPE", "2")
}
