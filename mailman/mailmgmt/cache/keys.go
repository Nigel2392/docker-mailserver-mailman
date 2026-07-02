package cache

import (
	"crypto/md5"
	"fmt"
	"net/http"
)

func RequestBasedCacheKey(request *http.Request, prefix string) string {
	var hash = md5.New()
	hash.Write([]byte(request.URL.String()))
	if prefix != "" {
		return fmt.Sprintf("%s.%x", prefix, hash.Sum(nil))
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
