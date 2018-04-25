package fly

import (
	"github.com/golang/groupcache"
	"net/url"
	"net/http/httputil"
)

type State struct {
	Downloading int64 `json:"downloading"`
	Downloaded  int64 `json:"downloaded"`
}

type HttpError struct {
	errCode int
	msg     []byte
}

func (e HttpError) Error() string {
	return string(e.msg)
}

type Fly struct {
	state  State
	config Config

	originURL      *url.URL
	globalCache    *groupcache.Group
	dynamic        *httputil.ReverseProxy
	proxySuffixArr []string
}

type ServerState struct {
	DownloadState  State                 `json:"download_state"`
	HotCacheState  groupcache.CacheStats `json:"hot_cache_state"`
	MainCacheState groupcache.CacheStats `json:"main_cache_state"`
}
