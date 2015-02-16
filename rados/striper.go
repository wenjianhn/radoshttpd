package rados
/* vim: set ts=4 shiftwidth=4 smarttab noet : */

// #cgo LDFLAGS: -lrados
// #include <stdlib.h>
// #include <rados/librados.h>
// #include <radosstriper/libradosstriper.h>
import "C"

import "unsafe"

type StriperPool struct {
    striper C.rados_striper_t
}

type AioCompletion struct {
    completion C.rados_completion_t
}

func (sp *StriperPool) Read(oid string, data []byte, offset uint64) (int, error) {
  if len(data) == 0 {
        return 0,nil
  }

  c_oid := C.CString(oid)
  defer C.free(unsafe.Pointer(c_oid))

  ret := C.rados_striper_read(sp.striper, c_oid,
                               (*C.char)(unsafe.Pointer(&data[0])),
                               C.size_t(len(data)),
                               C.uint64_t(offset))
  if ret >= 0 {
        return int(ret), nil
  } else {
    return 0, RadosError(int(ret))
  }

}

func (sp *StriperPool) Destroy() {
    C.rados_striper_destroy(sp.striper);
}

func (sp *StriperPool) State(oid string) (uint64, error) {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))
    var c_psize C.uint64_t
    ret := C.rados_striper_stat(sp.striper, c_oid, &c_psize, nil)
    if ret < 0 {
      return 0, RadosError(int(ret))
    }
    return uint64(c_psize), nil
}

func (sp *StriperPool) Delete(oid string) error {
    c_oid := C.CString(oid)
    defer C.free(unsafe.Pointer(c_oid))
    ret := C.rados_striper_remove(sp.striper, c_oid)
    if ret < 0 {
      return RadosError(int(ret))
    }
    return nil
}

func (sp *StriperPool) Write(oid string, data []byte, offset uint64) (int, error) {
  if len(data) == 0 {
        return 0,nil
  }

  c_oid := C.CString(oid)
  defer C.free(unsafe.Pointer(c_oid))

  ret := C.rados_striper_write(sp.striper, c_oid,
                               (*C.char)(unsafe.Pointer(&data[0])),
                               C.size_t(len(data)),
                               C.uint64_t(offset))
  if ret >= 0 {
        return int(ret), nil
  } else {
    return 0, RadosError(int(ret))
  }

}


func (sp *StriperPool) WriteAIO(c *AioCompletion, oid string, data []byte, offset uint64) (int, error) {
	if len(data) == 0 {
		return 0,nil
	}

	c_oid := C.CString(oid)

	ret := C.rados_striper_aio_write(sp.striper, c_oid, c.completion, (*C.char)(unsafe.Pointer(&data[0])),  C.size_t(len(data)), C.uint64_t(offset))
	if ret >= 0 {
		return int(ret), nil
	} else {
		return 0, RadosError(int(ret))
	}

}

func (sp *StriperPool) Flush() {
	C.rados_striper_aio_flush(sp.striper)
}

func (c *AioCompletion) Create() error {
	ret := C.rados_aio_create_completion(nil, nil, nil, (*C.rados_completion_t)(&c.completion))
	if ret >= 0 {
		return nil
	} else {
		return RadosError(int(ret))
	}
}

func (c *AioCompletion) WaitForComplete() {
	C.rados_aio_wait_for_complete(c.completion)
}

func (c *AioCompletion) Release() {
	C.rados_aio_release(c.completion)
}

func (c *AioCompletion) IsComplete() int{
	ret := int(C.rados_aio_is_complete(c.completion))
	return ret
}

func (c *AioCompletion) GetReturnValue() int {
	ret := int(C.rados_aio_get_return_value(c.completion))
	return ret
}
