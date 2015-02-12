package main

/*
#cgo LDFLAGS: -lssl -lcrypto
#include <openssl/md5.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

type MD5Context C.MD5_CTX

func MD5New() (*MD5Context, error) {
	var ctx MD5Context
	ret := C.MD5_Init((*C.MD5_CTX)(&ctx))
	if ret == 0 {
		return nil, errors.New("Error during MD5 init")
	}
	return &ctx, nil
}

func (ctx *MD5Context) Update(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	ret := C.MD5_Update((*C.MD5_CTX)(ctx), unsafe.Pointer(&data[0]), C.size_t(len(data)))
	if ret == 0 {
		return errors.New("Error during MD5 update")
	}

	return nil
}

func (ctx *MD5Context) Final() ([]byte, error) {
	res := make([]byte, C.MD5_DIGEST_LENGTH)
	ret := C.MD5_Final((*C.uchar)(unsafe.Pointer(&res[0])), (*C.MD5_CTX)(ctx))
	if ret == 0 {
		return res, errors.New("Error during MD5 finalize")
	}

	return res, nil
}
