## fly
1. 简单的mirror加速服务
2. 使用 groupcache 进行缓存加速
3. 可配置缓存时间

## 使用
cd bin && mkdir tmp && go build fly.go && ./fly http://example.com/


```
Usage:
  fly http://expample.com/ [flags]
  fly [command]

Available Commands:
  help        Help about any command
  version     Print the version number of orcworker

Flags:
  -a, --Address string           监听地址 (default ":9090")
  -e, --CacheExpireTime string   缓存过期时间,0为永不过期，单位：s,m,h,d, 例如:                                                                                                                                                                                                1h (default "0")
  -s, --CacheUriSuffix string    需要缓存的后缀名 (default "gif|jpg|jpeg|bmp|png                                                                                                                                                                                               |ico|txt|js|css|swf|ioc|rar|zip|flv|mid|doc|ppt|pdf|xls|mp3|wma")
  -c, --LocalCacheDir string     本地缓存地址 (default "./tmp")
  -l, --Logfile string           日志地址
  -m, --MaxMemCache string       缓存最大内存使用,单位： b,k,m,g，例如: 10m (def                                                                                                                                                                                               ault "10m")
  -h, --help                     help for fly

Use "fly [command] --help" for more information about a command.
```