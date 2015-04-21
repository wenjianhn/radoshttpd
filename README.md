# Cache

基于 [Groupcache](https://github.com/golang/groupcache) 实现。
每个wuzei节点既是Groupcache Server也是Groupcache Client.

Groupcache is a caching and cache-filling library, intended as a
replacement for memcached in many cases.

## 配置文件: /etc/wuzei/wuzei.json

给Groupcache用的选项如下：

* "Name": "objcache.wuzei"

	所有wuzei节点都应使用同样的名字，以共享Groupcache.

* "CacheSizeMBytes": 1024

	当前节点用于缓存的内存大小。单位为MiB.

	*注意*：

		受Golang GC的限制，wuzei释放的内存并不会及时还给OS。
		导致wuzei需要的最大内存为该缓存的2倍左右。
		经测试发现，当前最新的Golang 1.4 还是没有解决这个问题。

* "CacheChunkSizeKBytes": 2048

	cache分片的大小。单位为KiB.

* "CacheMaxObjectSizeKBytes": 4096

	最大可缓存对象的大小。单位为KiB.

	超过这个大小的对象不会被缓存。

* "MyIPAddr": "127.0.0.1"

	Groupcache Server监听这个地址。通过这个地址共享缓存给其他节点。

* "Port": 10946

	Groupcache Server监听的端口。

* "Peers": ["127.0.0.1"]

	所有Groupcache节点的IP地址。

	新增或者删除wuzei节点后，需要改变所有节点的这个配置。

改变配置文件中的MyIPAddr, Port和Peers后，可发SIGHUP给wuzei，重新读取配置文件。
