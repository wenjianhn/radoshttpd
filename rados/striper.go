package rados

// #cgo LDFLAGS: -lrados
// #include <stdlib.h>
// #include <rados/librados.h>
// #include <radosstriper/libradosstriper.h>
import "C"

import "unsafe"

type StriperPool struct {
    striper C.rados_striper_t
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
