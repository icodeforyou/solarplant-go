package www

import (
	"net/url"
	"strconv"
)

func intOrDefault(u *url.URL, key string, defaultValue int) int {
	if v := u.Query().Get(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
