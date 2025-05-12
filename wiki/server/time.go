package server

import (
	"net/http"
	"time"
)

// TimeLocationKey is the key for the *time.Location in the request context.
// When using mainLogin, this key will be set to the *time.Location set by the user.
var TimeLocationKey = "timeLocation"

func getTimeLocation(r *http.Request) *time.Location {
	loc, ok := r.Context().Value(TimeLocationKey).(*time.Location)
	if !ok {
		return time.UTC
	}
	return loc
}
