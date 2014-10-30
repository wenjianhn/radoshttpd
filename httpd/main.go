package main
import (
    "github.com/thesues/radoshttpd/rados"
    "github.com/codegangsta/martini"
    "fmt"
    "errors"
    "os"
    "net/http"
    "io"
)

/* this getData function is to test download remote rados file */
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

type RadosDownloader struct {
  striper *rados.StriperPool
  soid string
  offset uint64
}

func (rd *RadosDownloader) Read(p []byte) (n int, err error) {
  count, err := rd.striper.Read(rd.soid, p, uint64(rd.offset))
  if count == 0 {
    return 0, io.EOF
  }
  rd.offset += uint64(count)
  return count, err
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

    /* start from here */
    /*
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
    */

    /*
    err= getData(striper, "go", "dat.0")
    if err != nil {
        fmt.Println("error")
    }
    */

    m := martini.Classic()
     m.Get("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[A-Za-z0-9]+)", func(params martini.Params, w http.ResponseWriter, r *http.Request){
         poolname := params["pool"]
         soid := params["soid"]
         pool, err := conn.OpenPool(poolname)
         if err != nil {
           /* FIXME */
           fmt.Println("open pool failed")
           return
         }
         defer pool.Destroy()

         striper, err := pool.CreateStriper()
         if err != nil {
           fmt.Println("open pool failed")
           return
         }
         defer striper.Destroy()


         rd := RadosDownloader{&striper, soid, 0}
         /* set content-type */
         w.Header().Set("Content-Disposition", "attachment; filename=Wiki.png")
         w.Header().Set("Content-Type", r.Header.Get("Content-Type"))

         /* set the stream */
         io.Copy(w,&rd)
     })

    /* end */
    http.ListenAndServe(":3000", m)
}
