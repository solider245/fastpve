//go:build HAS_REMOTE_URL
// +build HAS_REMOTE_URL

package main

import (
	"time"

	"github.com/kspeeder/urlcache/remoteurlcache"
	"github.com/solider245/fastpve/downloader"
)

const (
	defaultCacheServerBaseURL = remoteurlcache.ServerURL
	defaultCacheAuthHeader    = remoteurlcache.AuthHeader
	defaultCacheTimeout       = 15 * time.Second
)

func init() {
	downloader.RegisterRemoteURLCacheProvider(func() downloader.RemoteURLCache {
		return remoteurlcache.New(defaultCacheServerBaseURL, defaultCacheAuthHeader, defaultCacheTimeout)
	})
}
