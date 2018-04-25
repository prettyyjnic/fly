package fly

type Config struct {
	MaxMemCacheBytes     int64  // 内存缓存大小
	Origin               string // 原站地址
	Logfile              string // log 地址
	Address              string // 监听地址
	LocalCacheDir        string // 本地缓存地址
	CacheUriSuffix       string // 需要缓存的后缀
}
