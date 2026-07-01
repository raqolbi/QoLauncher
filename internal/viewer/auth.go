package viewer

import (
	"crypto/subtle"
	"net/http"
)

const authRealm = "QoLauncher Log Viewer"

func basicAuth(username, password string, next http.Handler) http.Handler {
	expectedUser := []byte(username)
	expectedPass := []byte(password)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), expectedUser) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), expectedPass) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+authRealm+`"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
