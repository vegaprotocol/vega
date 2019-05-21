package gateway

import (
	"context"
	"net"
	"net/http"
	"strings"

	"code.vegaprotocol.io/vega/internal/logging"
)

const bearerPrefix = "Bearer "

type tokenKeyTy int

var tokenKey tokenKeyTy

func TokenFromContext(ctx context.Context) string {
	u, _ := ctx.Value(tokenKey).(string)
	return u
}

func TokenMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if authhdr := r.Header.Get("Authorization"); len(authhdr) > 0 {
			if strings.HasPrefix(authhdr, bearerPrefix) {
				tkn := strings.TrimPrefix(authhdr, bearerPrefix)
				r = r.WithContext(context.WithValue(r.Context(), "token", tkn))
			} else {
				log.Debug("token specified but invalid fmt",
					logging.String("remote-addr", r.RemoteAddr),
				)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func RemoteAddrMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		found := false
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Warn("Remote address is not splittable in middleware",
				logging.String("remote-addr", r.RemoteAddr))
		} else {
			userIP := net.ParseIP(ip)
			if userIP == nil {
				log.Warn("Remote address is not IP:port format in middleware",
					logging.String("remote-addr", r.RemoteAddr))
			} else {
				found = true

				// Only defined when site is accessed via non-anonymous proxy
				// and takes precedence over RemoteAddr
				forward := r.Header.Get("X-Forwarded-For")
				if forward != "" {
					ip = forward
				}
			}
		}

		if found {
			r = r.WithContext(context.WithValue(r.Context(), "remote-ip-addr", ip))
		}
		next.ServeHTTP(w, r)
	})
}
