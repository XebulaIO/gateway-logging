package main

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/proxy"
)

func JWTDecoderMiddleware(next proxy.Proxy) proxy.Proxy {
	return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
		authHeaders, ok := req.Headers["Authorization"]
		if ok && len(authHeaders) > 0 {
			bearerToken := strings.Split(authHeaders[0], " ")
			if len(bearerToken) == 2 {
				token, _ := jwt.Parse(bearerToken[1], nil)
				if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
					if sub, ok := claims["sub"].(string); ok {
						req.Headers["sub"] = []string{sub}
					}
				}
			}
		}
		return next(ctx, req)
	}
}

func Register(cfg *config.EndpointConfig) proxy.Middleware {
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic("Too many proxies")
		}
		if len(next) < 1 {
			panic("No proxy")
		}
		return JWTDecoderMiddleware(next[0])
	}
}
