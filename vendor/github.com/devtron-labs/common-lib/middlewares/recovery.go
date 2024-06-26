package middlewares

import (
	"encoding/json"
	"github.com/devtron-labs/common-lib/constants"
	"github.com/devtron-labs/common-lib/pubsub-lib/metrics"
	"log"
	"net/http"
	"runtime/debug"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			err := recover()
			if err != nil {
				metrics.IncPanicRecoveryCount("handler", r.Host, r.Method, r.RequestURI)
				log.Print(constants.PanicLogIdentifier, "recovered from panic", "err", err, "stack", string(debug.Stack()))
				jsonBody, _ := json.Marshal(map[string]string{
					"error": "internal server error",
				})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
