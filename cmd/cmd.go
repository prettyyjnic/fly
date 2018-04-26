package cmd

import (
	"github.com/spf13/cobra"
	"github.com/prettyyjnic/fly"
	"os"
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"strconv"
)

const VERSION = 1.0

var config fly.Config
var maxMemCache string
var cacheExpireTime string

var rootCmd = &cobra.Command{
	Use:   "fly http://expample.com/",
	Short: "简单的cnd服务",
	//TraverseChildren: true,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("请输入源站地址")
		}
		config.Origin = args[0]

		var err error
		var tmp int64
		config.LocalCacheDir = strings.Replace(config.LocalCacheDir, "\\", "/", -1)
		config.LocalCacheDir = strings.TrimRight(config.LocalCacheDir, "/") + "/"
		err = checkCacheDir(config.LocalCacheDir)
		if err != nil {
			return err
		}

		maxMemCache = strings.ToLower(maxMemCache)
		unit := maxMemCache[len(maxMemCache)-1]
		tmp, err = strconv.ParseInt(maxMemCache[:len(maxMemCache)-1], 10, 64)
		if err != nil {
			return err
		}
		switch unit {
		case 'b':
			config.MaxMemCacheBytes = tmp
		case 'k':
			config.MaxMemCacheBytes = tmp * 1024
		case 'm':
			config.MaxMemCacheBytes = tmp * 1024 * 1024
		case 'g':
			config.MaxMemCacheBytes = tmp * 1024 * 1024 * 1024
		}

		if cacheExpireTime == "0" {
			config.CacheExpireTime = 0
		}else{
			cacheExpireTimeUnit :=  cacheExpireTime[len(cacheExpireTime)-1]
			tmp, err = strconv.ParseInt(cacheExpireTime[:len(cacheExpireTime)-1], 10, 64)
			if err != nil {
				return err
			}
			switch cacheExpireTimeUnit {
			case 's':
				config.CacheExpireTime = tmp
			case 'm':
				config.CacheExpireTime = tmp * 60
			case 'h':
				config.CacheExpireTime = tmp * 60 * 60
			case 'd':
				config.CacheExpireTime = tmp * 60 * 60 * 24
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fly.Start(config)
	},
}

//版本信息
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of orcworker",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.Flags().StringVarP(&config.Logfile, "Logfile", "l", "", "日志地址")
	rootCmd.Flags().StringVarP(&config.Address, "Address", "a", ":9090", "监听地址")
	rootCmd.Flags().StringVarP(&config.LocalCacheDir, "LocalCacheDir", "c", "./tmp", "本地缓存地址")
	rootCmd.Flags().StringVarP(&maxMemCache, "MaxMemCache", "m", "10m", "缓存最大内存使用,单位： b,k,m,g，例如: 10m")
	rootCmd.Flags().StringVarP(&config.CacheUriSuffix, "CacheUriSuffix", "s", "gif|jpg|jpeg|bmp|png|ico|txt|js|css|swf|ioc|rar|zip|flv|mid|doc|ppt|pdf|xls|mp3|wma", "需要缓存的后缀名")

	rootCmd.Flags().StringVarP(&cacheExpireTime, "CacheExpireTime", "e", "0", "缓存过期时间,0为永不过期，单位：s,m,h,d, 例如: 1h")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// 检查缓存文件夹
func checkCacheDir(dir string) error {
	var err error
	var stat os.FileInfo
	stat, err = os.Stat(dir)
	if (err != nil && os.IsNotExist(err) ) || !stat.IsDir() {
		return errors.Errorf("缓存文件夹[ %s ]不存在", dir)
	}
	testFilename := dir + "/__cache_test.tmp"
	if fly.Write2disk(testFilename, []byte{1}) != nil {
		return errors.Errorf("缓存文件夹写入失败 %s", err.Error())
	}
	os.Remove(testFilename)
	return nil
}
