// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	logrustash "github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type JwtClaims struct {
	Exp          int64  `json:"exp"`
	Iat          int64  `json:"iat"`
	Jti          string `json:"jti"`
	Iss          string `json:"iss"`
	Aud          string `json:"aud"`
	Sub          string `json:"sub"`
	Typ          string `json:"typ"`
	Azp          string `json:"azp"`
	SessionState string `json:"session_state"`
	RealmAccess  struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess struct {
		Account struct {
			Roles []string `json:"roles"`
		} `json:"account"`
	}
	Scope             string `json:"scope"`
	Sid               string `json:"sid"`
	EmailVerified     bool   `json:"email_verified"`
	UserType          string `json:"user_type"`
	UserID            string `json:"user_id"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Email             string `json:"email"`
}

func (c *JwtClaims) Valid() error {
	return nil
}

// ClientRegisterer is the symbol the plugin loader will try to load. It must implement the RegisterClient interface
var (
	// client *elasticsearch.Client
	ClientRegisterer = registerer("xebula-logger")
	hook             logrus.Hook
)

func init() {
	var err error

	logrus.New()

	conn, err := net.Dial("tcp", "logstash-svc.elk-stack:5044")
	// conn, err := net.Dial("tcp", "host.docker.internal:5000")
	if err != nil {
		log.Fatalf("fluentd connection error: %v", err)
	}

	hook = logrustash.New(conn, logrustash.DefaultFormatter(logrus.Fields{"type": "xebula-logger"}))
	logrus.AddHook(hook)
}

type registerer string

var logger Logger = nil

func (registerer) RegisterLogger(v interface{}) {
	l, ok := v.(Logger)
	if !ok {
		return
	}
	logger = l
	logger.Debug(fmt.Sprintf("[PLUGIN: %s] Logger loaded", ClientRegisterer))
}

func (r registerer) RegisterClients(f func(
	name string,
	handler func(context.Context, map[string]interface{}) (http.Handler, error),
)) {
	f(string(r), r.registerClients)
}

func (r registerer) registerClients(_ context.Context, extra map[string]interface{}) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		rawData, err := io.ReadAll(req.Body)
		if err != nil {
			return
		}

		req.Body = io.NopCloser(strings.NewReader(string(rawData)))

		trackID := uuid.New().String()
		

		requestData := map[string]interface{}{
			"TrackID": trackID,
			"Body":    string(rawData),
			"Method":  req.Method,
			"Url":     req.URL.String(),
			"Query":   req.URL.RawQuery,
			"Path":    req.URL.Path,
		}

		
		realIP := req.Header.Get("Cf-Connecting-Ip")
		if realIP != "" {
			requestData["IP"] = realIP
		}

		accesToken := extractTokenFromHeader(req.Cookies(), "access_token")

		if accesToken != "" {
			token, _ := jwt.ParseWithClaims(accesToken, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte{}, nil
			})

			claims, ok := token.Claims.(*JwtClaims)
			if ok {
				requestData["UserID"] = claims.UserID
			}
		}

		err = sendLogToLogstash(requestData)
		if err != nil {
			logger.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rawData, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		resp.Body = io.NopCloser(strings.NewReader(string(rawData)))

		responseData := map[string]interface{}{
			"TrackID": trackID,
			"Url":     req.URL.String(),
			"Method":  req.Method,
			"Body":    string(rawData),
			"Status":  resp.StatusCode,
		}
		if realIP != "" {
			responseData["IP"] = realIP
		}

		if userID, ok := requestData["UserID"]; ok {
			responseData["UserID"] = userID
		}

		// Copy headers, status codes, and body from the backend to the response writer
		for k, hs := range resp.Header {
			for _, h := range hs {
				w.Header().Add(k, h)
			}
		}
		w.WriteHeader(resp.StatusCode)
		if resp.Body == nil {
			return
		}
		io.Copy(w, resp.Body)
		resp.Body.Close()

	}), nil
}
func sendLogToLogstash(d map[string]interface{}) error {
	jsonData, err := json.Marshal(d)
	if err != nil {
		return err
	}
	logrus.Info(string(jsonData))
	return nil
}

func extractTokenFromHeader(cookie []*http.Cookie, cookieName string) string {
	var value string

	for _, c := range cookie {
		if c.Name == cookieName {
			return c.Value
		}
	}

	return value
}

func main() {}

type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Critical(v ...interface{})
	Fatal(v ...interface{})
}
