package wugui

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/groupcache"
	"github.com/thesues/radoshttpd/rados"
)

var (
	chunkPool  sync.Pool
	radosCache *groupcache.Group
	cacheMade  bool
)

func GetRadosCacheStats() groupcache.Stats {
	return radosCache.Stats
}

func InitRadosCache(name string, size int64, chunkSize int) {
	if cacheMade {
		panic("groupcache: InitRadosCache must be called only once")
	}
	cacheMade = true

	radosCache = groupcache.NewGroup(name, size, groupcache.GetterFunc(radosGetter))
	chunkPool = sync.Pool{
		New: func() interface{} { return make([]byte, chunkSize) },
	}
}

type Key struct {
	poolname string
	filename string
	offset   int64
}

func (k *Key) FromString(key string) (err error) {
	i := strings.LastIndex(key, "-")
	k.offset, err = strconv.ParseInt(key[(i+1):], 10, 64)
	if err != nil {
		return err
	}

	blobref := key[:i]
	poolAndFile := strings.SplitN(blobref, "/", 2)
	k.poolname = poolAndFile[0]
	k.filename = poolAndFile[1]
	return
}

func (k *Key) String() string {
	// Bucket names cannot contain slash.
	// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html

	return fmt.Sprintf("%s/%s-%d", k.poolname, k.filename, k.offset)
}

type RadosReaderAt struct {
	striper  *rados.StriperPool
	poolname string
	filename string
	size     int64
}

func (r *RadosReaderAt) Size() int64 {
	return r.size
}

func (r *RadosReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	wantN := len(p)

	key := &Key{r.poolname, r.filename, off}
	err = radosCache.Get(r, key.String(),
		groupcache.TruncatingByteSliceSink(&p))
	if err != nil {
		return -1, err
	}

	if len(p) < wantN {
		return -1, io.ErrUnexpectedEOF
	}

	return len(p), err
}

// TODO(wenjianhn):
//  1. func NewRadosReaderAt(poolname string, filename string)
//  2. reuse rados.NewConn
func NewRadosReaderAt(striper *rados.StriperPool, poolname string, filename string, size int64) RadosReaderAt {
	return RadosReaderAt{
		striper:  striper,
		poolname: poolname,
		filename: filename,
		size:     size,
	}
}

func radosGetter(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	rr := ctx.(*RadosReaderAt)

	k := &Key{}
	err := k.FromString(key)
	if err != nil {
		return err
	}

	buf := chunkPool.Get().([]byte)
	defer chunkPool.Put(buf)
	readP := buf
	readN := 0
	off := k.offset
	for readN < cap(buf) {
		count, err := rr.striper.Read(rr.filename, readP, uint64(off))
		if err != nil {
			return fmt.Errorf("Timeout or Read Error: %s", err.Error())
		}
		if count == 0 {
			break
		}
		readN += count
		off += int64(count)
		readP = buf[count:]
	}
	return dest.SetBytes(buf[:readN])
}
