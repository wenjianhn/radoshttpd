/* GPLv3 */
/* deanraccoon@gmail.com */
/* vim: set ts=4 shiftwidth=4 smarttab noet : */

package main

import (
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/hydrogen18/stoppableListener"
	"github.com/thesues/radoshttpd/rados"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"errors"
	"strings"
	"strconv"
)

var (
	LOGPATH = "/var/log/wuzei/wuzei.log"
	PIDFILE = "/var/run/wuzei/wuzei.pid"
	slog    *log.Logger
	QUEUETIMEOUT time.Duration = 5 /* seconds */
	QUEUELENGTH = 100
	MONTIMEOUT = "30"
	OSDTIMEOUT = "30"
)

type RadosDownloader struct {
	striper *rados.StriperPool
	soid    string
	offset  int64
	buffer []byte
	waterhighmark int
	waterlowmark int
}


func min(a, b int) int {
   if a < b {
      return a
   }
   return b
}

func (rd *RadosDownloader) Read(p []byte) (n int, err error) {
	var count int = 0
	/* local buffer is empty */
	if (rd.waterhighmark == rd.waterlowmark) {
		count, err = rd.striper.Read(rd.soid, rd.buffer, uint64(rd.offset))
		/* Timeout or read error occurs */
		if err != nil {
			return count, errors.New("Timeout or Read Error")
		}
		if count == 0 {
			return 0, io.EOF
		}
		rd.offset += int64(count)
		rd.waterhighmark = count
		rd.waterlowmark = 0
	}

	l  := len(p)
	if l <= rd.waterhighmark - rd.waterlowmark {
		copy(p, rd.buffer[rd.waterlowmark:rd.waterlowmark + l])
		rd.waterlowmark += l
		return l, nil
	} else {
		copy(p, rd.buffer[rd.waterlowmark:rd.waterhighmark])
		rd.waterlowmark = rd.waterhighmark
		return rd.waterhighmark - rd.waterlowmark, nil
	}

	return 0, errors.New("read failed")
}


func (rd *RadosDownloader) Seek(offset int64, whence int) (int64, error) {
	switch whence{
	case 0:
		rd.offset = offset
		return offset, nil
	case 1:
		rd.offset += offset
		return rd.offset, nil
	case 2:
		size, err := rd.striper.State(rd.soid)
		if err != nil {
			return 0, nil
		}
		rd.offset = int64(size)
		return rd.offset, nil
	default:
		return 0, errors.New("failed to seek")
	}

}

func main() {
	/* pid */
	if err := CreatePidfile(PIDFILE); err != nil {
		fmt.Printf("can not create pid file %s\n", PIDFILE)
		return
	}
	defer RemovePidfile(PIDFILE)

	/* log  */
	f, err := os.OpenFile(LOGPATH, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("failed to open log\n")
		return
	}
	defer f.Close()

	m := martini.Classic()
	slog = log.New(f, "[wuzei]", log.LstdFlags)
	m.Map(slog)

	conn, err := rados.NewConn("admin")
	if err != nil {
		slog.Println("failed to open keyring")
		return
	}

	conn.SetConfigOption("rados_mon_op_timeout", MONTIMEOUT)
	conn.SetConfigOption("rados_osd_op_timeout", OSDTIMEOUT)

	err = conn.ReadConfigFile("/etc/ceph/ceph.conf")
	if err != nil {
		slog.Println("failed to open ceph.conf")
		return
	}


	err = conn.Connect()
	if err != nil {
		slog.Println("failed to connect to remote cluster")
		return
	}
	defer conn.Shutdown()

	var wg sync.WaitGroup

	m.Get("/whoareyou", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("I AM WUZEI"))
	})

	var ProcessSlots = make(chan bool, QUEUELENGTH)

	releaseSlot := func(){
		<-ProcessSlots
	}

	/* resume upload protocal is based on http://www.grid.net.ru/nginx/resumable_uploads.en.html */
	m.Put("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", func(params martini.Params, w http.ResponseWriter, r *http.Request) {
		/* prepare striprados */
		wg.Add(1)
		defer wg.Done()
		select {
		case <- time.After(QUEUETIMEOUT * time.Second):
			/* send timeout to client*/
			slog.Println("request timeout")
			ErrorHandler(w, r, http.StatusRequestTimeout)
			return
		case ProcessSlots <- true:
			/* write to channel to get a slot for writing rados object */
		}
		defer releaseSlot()

		poolname := params["pool"]
		soid := params["soid"]
		pool, err := conn.OpenPool(poolname)
		if err != nil {
			slog.Println("open pool failed")
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		defer pool.Destroy()
		striper, err := pool.CreateStriper()
		if err != nil {
			slog.Println("open pool failed")
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		defer striper.Destroy()

		bytesRange := r.Header.Get("Content-Range")

		var offset, start, end, size int64 = 0,0,0,0

		if bytesRange != "" {
			/* header format: Content-Range:bytes 0-99/300 */
			/* remove bytes and space */
			bytesRange = strings.Trim(bytesRange, "bytes")
			bytesRange = strings.TrimSpace(bytesRange)

			o := strings.Split(bytesRange, "/")
			currentRange, s := o[0], o[1]


			o = strings.Split(currentRange, "-")

			/* o[0] is the start, o[1] is the end */
			start, err = strconv.ParseInt(o[0], 10, 64)
			if err != nil {
				slog.Printf("parse Content-Range failed %s", bytesRange);
				ErrorHandler(w, r, http.StatusBadRequest)
				return
			}
			end, err = strconv.ParseInt(o[1], 10, 64)
			if err != nil {
				slog.Printf("parse Content-Range failed %s", bytesRange);
				ErrorHandler(w, r, http.StatusBadRequest)
				return
			}

			size, err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				slog.Printf("parse Content-Range failed %s", bytesRange);
				ErrorHandler(w, r, http.StatusBadRequest)
				return
			}
			/* I know this is ugly, this is a workaround, because libstriprados sometimes can not set size attr */
			/* so I set object size before any real write */
			if (size < 3) {
				slog.Printf("wuzei does not support too small file, file byterange is %s", bytesRange);
				ErrorHandler(w, r, http.StatusBadRequest)
				return
			}
			/* "s" here is for futher debug */
			striper.Write(soid, []byte("s"), uint64(size) -2);
			/* above is ugly */
		}

		bufferSize := 4 << 20 /* 4M buffer */
		buf := make([]byte, bufferSize)

		if bytesRange != "" {
			/* already get $start, $end, $size */
			offset = start
		} else {
			offset = 0
		}

		for offset < end || bytesRange == "" {
			l, err := r.Body.Read(buf)
			if l == 0 {
				break;
			}
			if err != nil {
				slog.Printf("failed to read content from client url:%s", r.RequestURI)
				ErrorHandler(w, r, http.StatusBadRequest)
				return

			}

			/* for situation that actuall transfered data is larger than range */
			if bytesRange != "" {
				l = min(l, int(end - offset + 1))
			}

			_, err = striper.Write(soid, buf[:l], uint64(offset))
			if err != nil {
				slog.Printf("write to ceph timeout, url is:%s", r.RequestURI)
				ErrorHandler(w, r, http.StatusRequestTimeout)
				return
			}
			offset += int64(l)
		}

		w.Header().Set("Content-Type", "application/octet-stream")

		if bytesRange == "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		/* support for bytesRange */
		if size == offset + 1 {
			/* return code 200 */
			w.WriteHeader(http.StatusOK)
		} else {
			/* 'offset - 1' is the last byte */
			w.Header().Set("Range", fmt.Sprintf("%d-%d/%d", start, offset - 1,  size))
			w.WriteHeader(http.StatusCreated)
		}

	})

	m.Get("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", func(params martini.Params, w http.ResponseWriter, r *http.Request) {
		/* used for graceful stop */
		wg.Add(1)
		defer wg.Done()

		select {
		case <- time.After(QUEUETIMEOUT * time.Second):
			/* send timeout to client*/
			slog.Println("request timeout")
			ErrorHandler(w, r, http.StatusRequestTimeout)
			return
		case ProcessSlots <- true:
			/* write to channel to get a slot for writing rados object */
		}
		defer releaseSlot()


		poolname := params["pool"]
		soid := params["soid"]
		pool, err := conn.OpenPool(poolname)
		if err != nil {
			slog.Println("open pool failed")
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		defer pool.Destroy()

		striper, err := pool.CreateStriper()
		if err != nil {
			slog.Println("open pool failed")
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		defer striper.Destroy()

		filename := fmt.Sprintf("%s", soid)
		size, err := striper.State(soid)
		if err != nil {
			slog.Println("failed to get object " + soid)
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}

		/* use 4M buffer to read */
		buffersize := 4<<20
		rd := RadosDownloader{&striper, soid, 0, make([]byte, buffersize), 0, 0}

		/* set content-type */
		/* Content-Type would be others */
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		/* set the stream */
		http.ServeContent(w, r, filename, time.Now(), &rd)
	})

	m.Delete("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", func(params martini.Params, w http.ResponseWriter, r *http.Request) {
		/* used for graceful stop */
		wg.Add(1)
		defer wg.Done()
		poolname := params["pool"]
		soid := params["soid"]
		pool, err := conn.OpenPool(poolname)
		if err != nil {
			slog.Println("open pool failed")
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		defer pool.Destroy()

		striper, err := pool.CreateStriper()
		if err != nil {
			slog.Println("open pool failed")
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		defer striper.Destroy()

		err = striper.Delete(soid)
		if err != nil {
			slog.Printf("delete object %s/%s failed\n", poolname, soid)
			ErrorHandler(w, r, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	originalListener, err := net.Listen("tcp", ":3000")
	sl, err := stoppableListener.New(originalListener)

	server := http.Server{}
	http.HandleFunc("/", m.ServeHTTP)

	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGINT,
						syscall.SIGHUP,
						syscall.SIGQUIT,
						syscall.SIGTERM)

	go func() {
		server.Serve(sl)
	}()

	slog.Printf("Serving HTTP\n")
	select {
	case signal := <-stop:
		slog.Printf("Got signal:%v\n", signal)
	}
	sl.Stop()
	slog.Printf("Waiting on server\n")
	wg.Wait()
	slog.Printf("Server shutdown\n")
}

func ErrorHandler(w http.ResponseWriter, r *http.Request, status int) {
	switch status {
	case http.StatusNotFound:
		w.WriteHeader(status)
		w.Write([]byte("object not found"))
	case http.StatusRequestTimeout:
		w.WriteHeader(status)
		w.Write([]byte("server is too busy,timeout"))
	}
}
