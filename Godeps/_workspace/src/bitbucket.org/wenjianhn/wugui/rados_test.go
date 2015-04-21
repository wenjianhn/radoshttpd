package wugui

import (
	"os/exec"
	"reflect"
	"testing"

	"github.com/thesues/radoshttpd/rados"
)

func TestGroupCacheKey(t *testing.T) {
	key := Key{"video.letv.io", "001-你好-世界!.mp4", 0}
	var k Key
	k.FromString(key.String())
	if !reflect.DeepEqual(key, k) {
		t.Errorf("Failed to bring the key back. Actual: %v. Expected: %v", k, key)
	}
}

func getUUID(t *testing.T) string {
	out, err := exec.Command("uuidgen").Output()
	if err != nil {
		t.Fatalf("Failed to generate uuid: %v", err)
	}
	return string(out[:36])
}

func newRadosConn(t *testing.T, user string, cfgPath string) (c *rados.Conn) {
	var err error
	c, err = rados.NewConn(user)
	if err != nil {
		t.Fatalf("Failed to open keyring: %v", err)
	}

	err = c.ReadConfigFile(cfgPath)
	if err != nil {
		t.Fatalf("Failed to read: %s", cfgPath)
	}

	err = c.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to remote cluster: %v", err)
	}
	return
}

func createStriperPool(t *testing.T, conn *rados.Conn, name string) rados.StriperPool {
	err := conn.MakePool(name)
	if err != nil {
		t.Fatalf("Failed to MakePool: %v", err)
	}

	// pool.Destroy() will not be called. It should be fine though.
	pool, err := conn.OpenPool(name)
	if err != nil {
		t.Fatalf("Failed to open pool %s: %v", name, err)
	}

	striper, err := pool.CreateStriper()
	if err != nil {
		t.Fatalf("Failed to create striper: %v", err)
	}
	return striper
}

func TestRados(t *testing.T) {
	me := "127.0.0.1"
	peers := []string{"127.0.0.1"}
	port := 16403
	InitCachePool(me, peers, port)
	InitRadosCache("s3.example.com", 1024, 4)

	conn := newRadosConn(t, "admin", "/etc/ceph/ceph.conf")
	defer conn.Shutdown()

	poolname := "TestRados_" + getUUID(t)
	striper := createStriperPool(t, conn, poolname)
	defer striper.Destroy()

	bytes_in := []byte("0123" + "4567" + "8")
	filename := "0to8"
	_, err := striper.Write(filename, bytes_in, 0)
	if err != nil {
		t.Errorf("Failed to Write: %v", err)
		return
	}

	size := int64(len(bytes_in))
	rr := NewRadosReaderAt(&striper, poolname, filename, size)
	readall := func() {
		buf := make([]byte, 4)
		var off, n int
		for left := size; left > 0; left -= int64(n) {
			readP := buf
			if left < 4 {
				readP = buf[:left]
			}
			n, err = rr.ReadAt(readP, int64(off))
			if err != nil {
				t.Errorf("rr.ReadAt failed: %v", err)
				return
			}
			if !reflect.DeepEqual(bytes_in[off:(off+n)], readP) {
				t.Errorf("Want: %q, got: %q",
					bytes_in[off:(off+n)], readP)
				return
			}
			off += n
		}
	}
	readall()
	stats := GetRadosCacheStats()
	if stats.Gets != 3 {
		t.Errorf("Cache Gets is wrong. Want: %d, got: %d", 3, stats.Gets)
		return
	}

	readall()
	stats = GetRadosCacheStats()
	if stats.CacheHits != 3 {
		t.Errorf("Cache Hits is wrong. Want: %d, got: %d", 3, stats.Gets)
	}
}
