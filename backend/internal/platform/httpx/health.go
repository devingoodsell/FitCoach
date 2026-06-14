package httpx

import "net/http"

// Health returns a handler for liveness checks. It reports 200 with a small JSON
// body and never touches dependencies, so it stays green during DB outages.
func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
