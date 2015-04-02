package rados


// #cgo CPPFLAGS: -I/usr/local/include -I./
// #cgo LDFLAGS: -L /usr/local/lib/ -lcls_stripesha1_client
// #include <stdlib.h>
// #include "radosstriper/cls_stripesha1_client.h"
import "C"
import "unsafe"


func (p *Pool) GetStripeSHA1(oid string) ([]byte, uint64, uint64, error) {
	c_oid := C.CString(oid)
	defer C.free(unsafe.Pointer(c_oid))
	var c_buflen C.int
	var c_piece_length C.uint64_t
	var c_length C.uint64_t
	var c_buf *C.char


	ret := C.cls_client_stripesha1_get(p.ioctx, c_oid, &c_buf, &c_buflen, &c_piece_length, &c_length);
	if ret < 0 {
		return nil, 0,0, RadosError(int(ret))
	}
	defer C.free(unsafe.Pointer(c_buf))
	//copy c_buf, c_buflen to data
	data := C.GoBytes(unsafe.Pointer(c_buf), c_buflen)
	return data, uint64(c_piece_length), uint64(c_length), nil

}
