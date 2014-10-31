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

/* copied from  io package */
/* default buf is too small for inner web */
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
   // If the reader has a WriteTo method, use it to do the copy.
   // Avoids an allocation and a copy.
   if wt, ok := src.(io.WriterTo); ok {
       return wt.WriteTo(dst)
   }
   // Similarly, if the writer has a ReadFrom method, use it to do the copy.
   if rt, ok := dst.(io.ReaderFrom); ok {
       return rt.ReadFrom(src)
   }
   buf := make([]byte, 32<<20)
   for {
       nr, er := src.Read(buf)
       if nr > 0 {
           nw, ew := dst.Write(buf[0:nr])
           if nw > 0 {
               written += int64(nw)
           }
           if ew != nil {
               err = ew
               break
           }
           if nr != nw {
               err = io.ErrShortWrite
               break
           }
       }
       if er == io.EOF {
           break
       }
       if er != nil {
           err = er
           break
       }
   }
   return written, err
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


    m := martini.Classic()
    m.Get("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[A-Za-z0-9-\\.]+)", func(params martini.Params, w http.ResponseWriter, r *http.Request){
         poolname := params["pool"]
         soid := params["soid"]
         /* FIXME */
         /* error checking */
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


         filename := fmt.Sprintf("%s-%s",poolname, soid)
         size, err :=  striper.State(soid)
         if err != nil {
           /* FIXME */
           size = 0
         }

         rd := RadosDownloader{&striper, soid, 0}
         /* set content-type */
         /* Content-Type would be others */
         w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
         w.Header().Set("Content-Length", fmt.Sprintf("%llu",size))
         w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))


         /* set the stream */
         Copy(w,&rd)
     })

    /* end */
    http.ListenAndServe(":3000", m)
}
