package wugui

import (
	"fmt"

	"github.com/golang/groupcache"
)

var httpPool *groupcache.HTTPPool

func addrToURL(addr []string, port int) []string {
	url := make([]string, len(addr))
	for i := range addr {
		url[i] = fmt.Sprintf("http://%s:%d", addr[i], port)
	}
	return url
}

func SetCachePoolPeers(peers []string, port int) {
	httpPool.Set(addrToURL(peers, port)...)
}

func InitCachePool(me string, peers []string, port int) {
	httpPool = groupcache.NewHTTPPool(fmt.Sprintf("http://%s:%d", me, port))

	SetCachePoolPeers(peers, port)
}
