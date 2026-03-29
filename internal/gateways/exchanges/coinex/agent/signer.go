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
	return &signer{credential: credential}
}

func (s *signer) signHttpGet(req *http.Request) {
	s.signHttp(req, time.Now().UnixMilli(), "")
}

func (s *signer) signHttpPost(req *http.Request, body []byte) {
	s.signHttp(req, time.Now().UnixMilli(), string(body))
}

func (s *signer) signHttp(req *http.Request, timestamp int64, data string) {
	apiKey := s.credential.GetApiKey()
	secret := s.credential.GetSecret()
	timestampStr := strconv.FormatInt(timestamp, 10)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(req.Method + req.URL.Path + data + timestampStr))

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-COINEX-KEY", apiKey)
	req.Header.Set("X-COINEX-TIMESTAMP", timestampStr)
	req.Header.Set("X-COINEX-SIGN", hex.EncodeToString(h.Sum(nil)))
}
