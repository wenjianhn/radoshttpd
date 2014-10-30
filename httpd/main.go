package main
import (
    "github.com/thesues/radoshttpd/rados"
    "fmt"
    "errors"
    "os"
)

func getData(striper rados.StriperPool, oid string, filename string) error{
    var offset int = 0
    var count  int = 0
    var err error
    buf := make([]byte, 32 << 20)
    f, err := os.Create(filename)
    if err != nil {
            return errors.New("can not create file")
    }
    defer f.Close()

    for {
        count, err = striper.Read(oid, buf, uint64(offset))
        if err != nil {
            return errors.New("download failed")
        }
        if count == 0 {
            break
        }
        f.WriteAt(buf[:count], int64(offset))

        offset += count;
        fmt.Println("%lu\r", offset);

   }
   return nil
}

func main() {
    conn, err := rados.NewConn("admin")
    if err != nil {
        return
    }
    conn.ReadConfigFile("/etc/ceph/ceph.conf");

    err = conn.Connect()
    if err != nil {
        return
    }
    defer conn.Shutdown()

    pool, err := conn.OpenPool("data")
    if err != nil {
        return
    }
    defer pool.Destroy()



    striper, err := pool.CreateStriper()
    if err != nil {
        return
    }
    defer striper.Destroy()

    err= getData(striper, "go", "dat.0")
    if err != nil {
        fmt.Println("error")
    }
}
