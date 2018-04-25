package fly

import (
	"log"
	"net/http"
	"github.com/golang/groupcache"
	"io/ioutil"
	"net/url"
	"time"
	"path/filepath"
	"bytes"
	"sync/atomic"
	"os"
	"encoding/json"
	"strings"
	"net/http/httputil"
	"path"
	"fmt"
)

const defaultPerm = os.FileMode(0755)

var logger *log.Logger

func Start(config Config) {
	fly := &Fly{
		config: config,
	}
	fly.server()
}

func (this *Fly) server() {
	if this.config.Logfile != "" {
		f, err := os.OpenFile(this.config.Logfile, os.O_CREATE|os.O_APPEND, 0755)
		if err != nil {
			log.Fatal("log 文件打开失败: ", err)
		}
		logger = log.New(f, "", log.Ldate|log.Ltime|log.Llongfile)
	} else {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Llongfile)
	}

	this.globalCache = groupcache.NewGroup("global", this.config.MaxMemCacheBytes, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			return this.Get(ctx, key, dest)
		}))
	this.originURL, _ = url.Parse(this.config.Origin)

	this.dynamic = httputil.NewSingleHostReverseProxy(this.originURL)
	this.proxySuffixArr = strings.Split(this.config.CacheUriSuffix, "|")
	http.HandleFunc("/__status", func(writer http.ResponseWriter, request *http.Request) {
		this.StatusHandle(writer, request)
	})
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		this.proxy(writer, request)
	})
	this.startTime = time.Now()
	var err error
	log.Println("服务启动", this.config.Address)
	err = http.ListenAndServe(this.config.Address, nil) //设置监听的端口
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
func (this Fly) StatusHandle(writer http.ResponseWriter, request *http.Request) {
	serverState := &ServerState{}
	serverState.DownloadState = this.state
	serverState.HotCacheState = this.globalCache.CacheStats(groupcache.HotCache)
	serverState.MainCacheState = this.globalCache.CacheStats(groupcache.MainCache)
	tmp, _ := json.Marshal(serverState)
	writer.Write(tmp)
}

func (this Fly) Get(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	if this.config.CacheExpireTime > 0 {
		key = this.decodeCacheKey(key) // 去掉缓存
	}
	filename := this.config.LocalCacheDir + strings.TrimLeft(key, "/")
	// 是否存在本地文件， 如果没有，则回源
	isExist, err := pathExists(filename)
	if err != nil {
		return err
	}
	// 获取文件修改时间
	modifyTime, err := getLastModifyTime(filename)
	if err != nil {
		return err
	}
	var bytesRead []byte
	if time.Now().Sub(modifyTime) > time.Second*time.Duration(this.config.CacheExpireTime) {
		goto Download
	}
	if isExist {
		bytesRead, err = ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		dest.SetBytes(bytesRead)
		return nil
	}
Download:
	atomic.AddInt64(&this.state.Downloading, 1)
	defer atomic.AddInt64(&this.state.Downloading, -1)
	bytesRead, err = this.proxyGet(key)
	if err != nil {
		return err
	}
	//写入到本地文件
	err = Write2disk(filename, bytesRead)
	if err != nil {
		logger.Printf("写入文件 %s 失败: %s", filename, err.Error())
	}
	atomic.AddInt64(&this.state.Downloaded, 1)
	dest.SetBytes(bytesRead)
	return nil
}

func (this *Fly) proxy(writer http.ResponseWriter, request *http.Request) {
	var isDynamic bool = true
	fpath := request.URL.Path
	if strings.ToLower(request.Method) == "get" { // 不是get方法的
		if fpath[len(fpath)-1] == '/' { // "/" 结尾的视为动态
			goto Server
		}
		ext := path.Ext(fpath)[1:]
		for i := 0; i < len(this.proxySuffixArr); i++ {
			if ext == this.proxySuffixArr[i] { // 后缀名符合
				isDynamic = false
				goto Server
			}
		}
	}
Server:
//动态资源
	if isDynamic {
		request.Host = this.originURL.Host //设置host，不然虚拟主机有bug
		this.dynamic.ServeHTTP(writer, request)
	} else {
		//如果是静态资源则
		this.staticProxy(writer, request)
	}
}
func (this *Fly) staticProxy(writer http.ResponseWriter, request *http.Request) {
	key := request.URL.Path
	if this.config.CacheExpireTime > 0 {
		key = this.genCacheKey(key)
	}
	request.Host = this.originURL.Host //设置host，不然虚拟主机有bug
	var data []byte
	var ctx groupcache.Context

	err := this.globalCache.Get(ctx, key, groupcache.AllocatingByteSliceSink(&data))
	if err != nil {
		if httpError, ok := err.(HttpError); ok {
			http.Error(writer, httpError.Error(), httpError.errCode)
		}
		http.Error(writer, err.Error(), 500)
		return
	}
	http.ServeContent(writer, request, filepath.Base(key), time.Now(), bytes.NewReader(data))
}
func (this Fly) proxyGet(key string) ([]byte, error) {
	u, _ := url.Parse(this.config.Origin)
	u.Path = key
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respStr, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 { // http错误
		return nil, &HttpError{errCode: resp.StatusCode, msg: respStr}
	}
	return respStr, err
}

func Write2disk(filename string, data []byte) error {
	var err error
	dirctory := filename[:strings.LastIndex(filename, "/")]
	isExist, err := pathExists(dirctory)
	if err != nil {
		return err
	}
	if !isExist {
		err = os.MkdirAll(dirctory, defaultPerm)
		if err != nil {
			log.Println("创建文件夹", dirctory, "失败！", err.Error())
			return err
		}
	}
	//写入文件
	return ioutil.WriteFile(filename, data, defaultPerm)
}

func pathExists(fpath string) (bool, error) {
	_, err := os.Stat(fpath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//生成缓存key
func (this *Fly) genCacheKey(key string) string {
	if this.config.CacheExpireTime == 0 {
		return key
	}
	nowSeconds := int64(time.Now().Sub(this.startTime).Seconds())
	return fmt.Sprintf("%d_%s", int64(nowSeconds/this.config.CacheExpireTime), key)
}

func (this *Fly) decodeCacheKey(key string) string {
	pos := strings.IndexByte(key, '_') + 1
	if pos == -1 {
		return key
	}
	return key[pos:]
}

func getLastModifyTime(filename string) (time.Time, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), nil
}
