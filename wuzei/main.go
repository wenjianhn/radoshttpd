/* GPLv3 */
/* deanraccoon@gmail.com */
/* vim: set ts=4 shiftwidth=4 smarttab noet : */

package main

import (
	"errors"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/thesues/radoshttpd/rados"
	"github.com/thesues/radoshttpd/nettimeout"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"encoding/hex"
	"container/list"
)

var (
	LOGPATH                    = "/var/log/wuzei/wuzei.log"
	PIDFILE                    = "/var/run/wuzei/wuzei.pid"
	QUEUETIMEOUT time.Duration = 5 /* seconds */
	QUEUELENGTH                = 100
	MONTIMEOUT                 = "30"
	OSDTIMEOUT                 = "30"
	BUFFERSIZE                 = 4 << 20 /* 4M */
	AIOCONCURRENT              = 4
	MAX_CHUNK_SIZE             = BUFFERSIZE * 2

	STRIPE_UNIT		   = uint(512 << 10) /* 512K */
	OBJECT_SIZE                = uint(64 << 20) /* 64M */
	STRIPE_COUNT               = uint(4)

	/* global variables */
	slog     *log.Logger
	conn     *rados.Conn
	ReqQueue RequestQueue
	wg       sync.WaitGroup
)

type RadosDownloader struct {
	striper       *rados.StriperPool
	soid          string
	offset        int64
	buffer        []byte
	waterhighmark int
	waterlowmark  int
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
	if rd.waterhighmark == rd.waterlowmark {
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

	l := len(p)
	if l <= rd.waterhighmark-rd.waterlowmark {
		copy(p, rd.buffer[rd.waterlowmark:rd.waterlowmark+l])
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
	switch whence {
	case 0:
		rd.offset = offset
		return offset, nil
	case 1:
		rd.offset += offset
		return rd.offset, nil
	case 2:
		size, _, err := rd.striper.State(rd.soid)
		if err != nil {
			return 0, nil
		}
		rd.offset = int64(size)
		return rd.offset, nil
	default:
		return 0, errors.New("failed to seek")
	}

}

/* RequestQueue has two functions */
/* 2. slot is used to queue write/read request */
type RequestQueue struct {
	slots chan bool
}

func (r *RequestQueue) Init() {
	r.slots = make(chan bool, QUEUELENGTH)
}

func (r *RequestQueue) inc() error {
	select {
	case <-time.After(QUEUETIMEOUT * time.Second):
		return errors.New("Queue is too long, timeout")
	case r.slots <- true:
		/* write to channel to get a slot for writing/reading rados object */
	}
	return nil
}

func (r *RequestQueue) dec() {
	<-r.slots
}

func GetHandler(params martini.Params, w http.ResponseWriter, r *http.Request) {

	/* used for graceful stop */
	wg.Add(1)
	defer wg.Done()
	if err := ReqQueue.inc(); err != nil {
		slog.Println("request timeout")
		ErrorHandler(w, r, http.StatusRequestTimeout)
		return
	}
	defer ReqQueue.dec()

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
	size, _, err := striper.State(soid)
	if err != nil {
		slog.Println("failed to get object " + soid)
		ErrorHandler(w, r, http.StatusNotFound)
		return
	}

	/* use 4M buffer to read */
	buffersize := BUFFERSIZE
	rd := RadosDownloader{&striper, soid, 0, make([]byte, buffersize), 0, 0}

	/* set let ServerContent to detect file type  */
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

	/* set the stream */
	http.ServeContent(w, r, filename, time.Now(), &rd)
}

func BlockHandler(params martini.Params, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("{\"blocksize\":%d}", MAX_CHUNK_SIZE * AIOCONCURRENT)))
}


func Md5sumHandler(params martini.Params, w http.ResponseWriter, r *http.Request) {
	/* used for graceful stop */
	wg.Add(1)
	defer wg.Done()

	if err := ReqQueue.inc(); err != nil {
		slog.Println("request timeout")
		ErrorHandler(w, r, http.StatusRequestTimeout)
		return
	}
	defer ReqQueue.dec()

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
	defer striper.Flush()


	var offset int64 = 0
	var start, end int64 = 0, 0
	var count,l int = 0, 0

	/* header format: Range:bytes 0-99 */
	bytesRange := r.Header.Get("Range")
	if bytesRange != "" {
		bytesRange = strings.Trim(bytesRange, "bytes")
		bytesRange = strings.TrimSpace(bytesRange)
		o := strings.Split(bytesRange, "-")
		start, err = strconv.ParseInt(o[0], 10, 64)
		if err != nil {
			slog.Printf("parse Content-Range failed %s", bytesRange)
			ErrorHandler(w, r, http.StatusBadRequest)
			return
		}
		end, err = strconv.ParseInt(o[1], 10, 64)
		if err != nil {
			slog.Printf("parse Content-Range failed %s", bytesRange)
			ErrorHandler(w, r, http.StatusBadRequest)
			return
		}
		offset = start
	}


	md5_ctx,_ := MD5New()
	buf := make([]byte, BUFFERSIZE)
	for offset <= end || bytesRange == "" {
		count, err = striper.Read(soid, buf, uint64(offset))
		if err != nil {
			slog.Printf("failed to read data for md5sum")
			ErrorHandler(w, r, 404)
			return
		}
		if count == 0 {
			break
		}

		/* Handle striper.read more data than expected when having Range Header*/
		if bytesRange != "" && offset + int64(count) > end {
			l = int(end - offset) + 1
		} else {
			l = count
		}

		if err = md5_ctx.Update(buf[:l]); err != nil {
			slog.Printf("calc md5sum failed")
			ErrorHandler(w, r, 501)
			return
		}
		offset += int64(count)
	}

	var b []byte
	if b, err = md5_ctx.Final(); err != nil {
		slog.Printf("calc md5sum failed")
		ErrorHandler(w, r, 501)
		return
	}

	s := hex.EncodeToString(b)
	w.Write([]byte(fmt.Sprintf("{\"md5\":\"%s\"}", s)))
}

func InfoHandler(params martini.Params, w http.ResponseWriter, r *http.Request) {
	/* used for graceful stop */
	wg.Add(1)
	defer wg.Done()

	if err := ReqQueue.inc(); err != nil {
		slog.Println("request timeout")
		ErrorHandler(w, r, http.StatusRequestTimeout)
		return
	}
	defer ReqQueue.dec()

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

	size, _, err := striper.State(soid)
	if err != nil {
		slog.Println("failed to get object " + soid)
		ErrorHandler(w, r, http.StatusNotFound)
		return
	}
	/* use json format */
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"size\":%d}", size)))
	return
}

func DeleteHandler(params martini.Params, w http.ResponseWriter, r *http.Request) {
	/* used for graceful stop */
	wg.Add(1)
	defer wg.Done()
	if err := ReqQueue.inc(); err != nil {
		slog.Println("request timeout")
		ErrorHandler(w, r, http.StatusRequestTimeout)
		return
	}
	defer ReqQueue.dec()

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
}

/* Function name 'pending_has_completed' and 'wait_pending_front' are the same as the radosgw  */
func pending_has_completed(p *list.List) bool{
	if p.Len() == 0 {
		return false
	}
	e := p.Front()
	c := e.Value.(*rados.AioCompletion)
	ret := c.IsComplete()
	if ret == 0 {
		return false
	} else {
		return true
	}
}


func wait_pending_front(p * list.List) int{
	/* remove AioCompletion from list */
	e := p.Front()
	p.Remove(e)
	c := e.Value.(*rados.AioCompletion)
	c.WaitForComplete()
	ret := c.GetReturnValue()
	c.Release()
	return ret
}


func drain_pending(p * list.List) int {
	var ret int
	for p.Len() > 0 {
		ret = wait_pending_front(p)
	}
	return ret
}


func set_stripe_layout(p * rados.StriperPool) int{
	var ret int = 0
	if ret = p.SetLayoutStripeUnit(STRIPE_UNIT); ret < 0 {
		return ret
	}
	if ret = p.SetLayoutObjectSize(OBJECT_SIZE); ret < 0 {
		return ret
	}
	if ret = p.SetLayoutStripeCount(STRIPE_COUNT); ret < 0 {
		return ret
	}
	return ret
}

func PutHandler(params martini.Params, w http.ResponseWriter, r *http.Request) {

	wg.Add(1)
	defer wg.Done()
	if err := ReqQueue.inc(); err != nil {
		slog.Println("request timeout")
		ErrorHandler(w, r, http.StatusRequestTimeout)
		return
	}
	defer ReqQueue.dec()

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
	set_stripe_layout(&striper)

	bytesRange := r.Header.Get("Content-Range")

	var src_offset, dest_offset, start, end, size int64 = 0, 0, 0, 0, 0

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
			slog.Printf("parse Content-Range failed %s", bytesRange)
			ErrorHandler(w, r, http.StatusBadRequest)
			return
		}
		end, err = strconv.ParseInt(o[1], 10, 64)
		if err != nil {
			slog.Printf("parse Content-Range failed %s", bytesRange)
			ErrorHandler(w, r, http.StatusBadRequest)
			return
		}

		size, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			slog.Printf("parse Content-Range failed %s", bytesRange)
			ErrorHandler(w, r, http.StatusBadRequest)
			return
		}
	}


	if bytesRange != "" {
		/* already get $start, $end */
		src_offset = start
		dest_offset = start
	} else {
		src_offset = 0
		dest_offset = 0
	}


	buf := make([]byte, BUFFERSIZE)
	/* if the data len in pending_data is bigger than MAX_CHUNK_SIZE, I will flush the data to ceph */
	var pending_data []byte
	var available_data_size int
	var c  *rados.AioCompletion
	pending := list.New()

	for src_offset <= end || bytesRange == "" {

		count, err := r.Body.Read(buf)
		if count == 0 {
			break
		}
		if err != nil {
			slog.Printf("failed to read content from client url:%s", r.RequestURI)
			drain_pending(pending)
			ErrorHandler(w, r, http.StatusBadRequest)
			return
		}

		//In case the user send more data than expected.
		if bytesRange != "" {
			available_data_size = min(count, int(end - src_offset + 1))
		} else {
			available_data_size = count
		}
		src_offset += int64(count)

		/* add newly received buffer to pending_data */
		pending_data = append(pending_data, buf[:available_data_size]...)

		/* if pending_data is not big enough, continue to read more data */
		if len(pending_data) < MAX_CHUNK_SIZE {
			continue
		}

		/* will write bl to ceph */
		bl := pending_data[:MAX_CHUNK_SIZE]
		/* now pending_data point to remaining data */
		pending_data = pending_data[MAX_CHUNK_SIZE:]


		c = new(rados.AioCompletion)
		c.Create()
		_, err = striper.WriteAIO(c, soid, bl, uint64(dest_offset))
		if err != nil {
			slog.Printf("starting to write aio failed")
			c.Release()
			drain_pending(pending)
			ErrorHandler(w, r, 501)
			return
		}
		pending.PushBack(c)

		//Throttle data
		//if the front is finished, cleanup 
		for pending_has_completed(pending) {
			if ret := wait_pending_front(pending); ret < 0 {
				slog.Printf("write aio failed or timeout, in pending_has_completed")
				drain_pending(pending)
				ErrorHandler(w, r, 408)
				return
			}
		}

		if pending.Len() > AIOCONCURRENT {
			slog.Println("inputstream is a bit faster, wait to finish")
			if ret := wait_pending_front(pending); ret < 0 {
				slog.Printf("write aio failed or timeout, in waiting pending ")
				drain_pending(pending)
				ErrorHandler(w, r, 408)
				return
			}
		}

		dest_offset += int64(len(bl))
	}

	//write all remaining data	
	if len(pending_data) > 0 {
		c = new(rados.AioCompletion)
		c.Create()
		striper.WriteAIO(c, soid, pending_data, uint64(dest_offset))
		pending.PushBack(c)
	}


	//drain_pending
	if ret := drain_pending(pending); ret < 0 {
		slog.Printf("write aio failed or timeout, in draining")
		ErrorHandler(w, r, 408)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")

	if bytesRange == "" {
		w.WriteHeader(http.StatusOK)
		return
	} else {
		/* user send too much data or right data*/
		if (src_offset >= end) {
			w.Header().Set("Range", fmt.Sprintf("%d-%d/%d", start, end, size))
		} else {
			/* user send too few data */
			w.Header().Set("Range", fmt.Sprintf("%d-%d/%d", start, src_offset - 1, size))
		}
		w.WriteHeader(http.StatusOK)
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

	conn, err = rados.NewConn("admin")
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

	ReqQueue.Init()

	m.Get("/whoareyou", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("I AM WUZEI"))
	})

	/* resume upload protocal is based on http://www.grid.net.ru/nginx/resumable_uploads.en.html */
	m.Put("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", PutHandler)
	m.Delete("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", DeleteHandler)
	m.Get("/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", GetHandler)
	m.Get("/info/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", InfoHandler)
	m.Get("/calcmd5/(?P<pool>[A-Za-z0-9]+)/(?P<soid>[^/]+)", Md5sumHandler)
	m.Get("/blocksize/",BlockHandler)

	sl,_ := nettimeout.NewListener("3000", 30* time.Second, 30 * time.Second);

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


/*
func parseContentRange(s string) (start ,end uint64) {

}

func parseRange(s string) {

}
*/

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
