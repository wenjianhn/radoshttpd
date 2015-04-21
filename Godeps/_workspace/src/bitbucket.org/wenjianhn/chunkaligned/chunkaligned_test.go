// Copyright 2015 Letv Cloud Computing. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package chunkaligned

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestNewChunkAlignedReaderAt(t *testing.T) {
	s := "0123" + "4567" + "8"
	sr := io.NewSectionReader(strings.NewReader(s), 0, int64(len(s)))
	r, err := NewChunkAlignedReaderAt(sr, 4)
	if err != nil {
		t.Fatalf("NewChunkAlignedReaderAt: %v", err)
	}

	if r.Size() != int64(len(s)) {
		t.Fatalf("Size() is %d, want %d", r.Size(), len(s))
	}

	buf := make([]byte, 9)
	for i := 1; i < len(buf); i++ {
		for off := 0; off < len(s); off++ {
			if (i + off) > len(s) {
				continue
			}
			readP := buf[:i]
			_, err = r.ReadAt(readP, int64(off))
			if err != nil {
				t.Fatalf("r.ReadAt: %v", err)
			}
			if !reflect.DeepEqual(s[off:off+len(readP)], string(readP)) {
				t.Errorf("Want %s, got %s", s[off:off+len(readP)], readP)
			}
		}
	}

	_, err = r.ReadAt(buf, int64(len(s)))
	if err != io.EOF {
		t.Fatalf("Want %v, got %v", io.EOF, err)
	}
}

func TestIntergration(t *testing.T) {
	tf, err := sizeTempFile(1 * 1024 * 1024)
	if err != nil {
		t.Errorf("sizeTempFile: %v", err)
		return
	}
	defer os.Remove(tf.Name())
	defer tf.Close()

	fg, err := newFileGetter(tf.Name())
	if err != nil {
		t.Errorf("newFileGetter: %v", err)
		return
	}
	defer fg.Close()

	// testing with different combinations of chunkSize, bufSize and offset
	for i := 0; i < 6; i++ {
		chunkSize := 32 * 1024 * (1 << uint(i))
		cara, err := NewChunkAlignedReaderAt(&fg, chunkSize)
		if err != nil {
			t.Errorf("NewChunkAlignedReaderAt: %v", err)
			return
		}
		for j := 0; j < 4; j++ {
			bufSize := 32 * 1024 * (1 << uint(j))
			for k := 0; k < 100; k++ {
				offset := rand.Int63n(fg.Size())

				t.Logf("chunkSize: %d, bufSize: %d, offset: %d",
					chunkSize, bufSize, offset)

				wantN := bufSize
				if fg.Size()-offset < int64(bufSize) {
					wantN = int(fg.Size() - offset)
				}

				bufActual := make([]byte, wantN)
				_, err = cara.ReadAt(bufActual, offset)
				if err != nil {
					t.Errorf("cara.ReadAt: %v", err)
					return
				}

				bufExpected := make([]byte, wantN)
				_, err = tf.ReadAt(bufExpected, offset)
				if err != nil {
					t.Errorf("tf.ReadAt: %v", err)
					return
				}

				if !reflect.DeepEqual(bufActual, bufExpected) {
					t.Errorf("ReadAt did not work properly")
					return
				}
			}
		}
	}
}

func TestChunkSizeLimitExceeded(t *testing.T) {
	s := "very large buffer"
	r := io.NewSectionReader(strings.NewReader(s), 0, int64(len(s)))

	_, err := NewChunkAlignedReaderAt(r, chunkSizeLimit+1)
	if err.Error() != "chunkaligned: chunk size limit exceeded" {
		t.Errorf("Limitting chunkSize didn't work properly")
	}
}

type fileGetter struct {
	size int64
	file *os.File
}

func (fg *fileGetter) Size() int64 {
	return fg.size
}

func (fg *fileGetter) ReadAt(p []byte, off int64) (n int, err error) {
	return fg.file.ReadAt(p, off)
}

func (fg *fileGetter) Close() error {
	return fg.file.Close()
}

func newFileGetter(path string) (fileGetter, error) {
	file, err := os.Open(path)
	if err != nil {
		return fileGetter{}, err
	}

	fs, err := file.Stat()
	if err != nil {
		return fileGetter{}, err
	}

	return fileGetter{fs.Size(), file}, nil
}

func sizeTempFile(size int64) (f *os.File, err error) {
	tf, err := ioutil.TempFile("", "_chunkaligned_")
	if err != nil {
		err = fmt.Errorf("TempFile: %v", err)
		return
	}

	rf, err := os.Open("/dev/urandom")
	if err != nil {
		err = fmt.Errorf("Open /dev/urandom: %v", err)
		return
	}
	defer rf.Close()

	_, err = io.CopyN(tf, rf, size)
	if err != nil {
		err = fmt.Errorf("Failed to write content: %v", err)
		return
	}

	return tf, nil
}
