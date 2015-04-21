// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Most of the contribution goes to Brad Fitzpatrick.
// See http://talks.golang.org/2013/oscon-dl.slide

package chunkaligned

import (
	"errors"
	"io"
	"math"
	"sort"
	"sync"
)

// An io.SectionReader implements SizeReaderAt.
type SizeReaderAt interface {
	Size() int64
	io.ReaderAt
}

type offsetAndSource struct {
	off int64
	SizeReaderAt
}

type multi struct {
	parts []offsetAndSource
	size  int64
}

func (m *multi) Size() int64 { return m.size }

func (m *multi) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= m.size {
		return 0, io.EOF
	}

	wantN := len(p)

	// Skip past the requested offset.
	skipParts := sort.Search(len(m.parts), func(i int) bool {
		// This function returns whether parts[i] will
		// contribute any bytes to our output.
		part := m.parts[i]
		return part.off+part.Size() > off
	})
	parts := m.parts[skipParts:]

	// How far to skip in the first part.
	needSkip := off
	if len(parts) > 0 {
		needSkip -= parts[0].off
	}

	for len(parts) > 0 && len(p) > 0 {
		readP := p
		partSize := parts[0].Size()
		if int64(len(readP)) > partSize-needSkip {
			readP = readP[:partSize-needSkip]
		}
		pn, err0 := parts[0].ReadAt(readP, needSkip)
		if err0 != nil {
			return n, err0
		}
		n += pn
		p = p[pn:]
		if int64(pn)+needSkip == partSize {
			parts = parts[1:]
		}
		needSkip = 0
	}

	if n != wantN {
		err = io.ErrUnexpectedEOF
	}
	return
}

const (
	chunkSizeLimit = 4 * 1024 * 1024
)

// fixed-length []byte pool, they will not grow as needed
var fixedBytePool = sync.Pool{
	New: func() interface{} { return make([]byte, chunkSizeLimit) },
}

// NOTE(wenjianhn): Clients of chunkReadAt cannot execute parallel
// ReadAt calls on the same chunk, beacause there is no lock to guard the cache.
type chunkReaderAt struct {
	size  int
	base  int64
	cache []byte
	r     SizeReaderAt
}

func (c *chunkReaderAt) Size() int64 {
	return int64(c.size)
}

func (c *chunkReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	wantN := len(p)

	if len(c.cache) == 0 {
		c.cache = fixedBytePool.Get().([]byte)

		// the offset is aligned
		readN, err := c.r.ReadAt(c.cache[:c.size], c.base)
		if err != nil {
			if err == io.EOF {
				if readN < wantN {
					fixedBytePool.Put(c.cache)
					c.cache = nil

					// We always know when EOF is coming.
					// If the caller asked for a chunk, there should be a chunk.
					return 0, io.ErrUnexpectedEOF
				}
			} else {
				fixedBytePool.Put(c.cache)
				c.cache = nil
				return 0, err
			}
		}
	}

	needSkip := int(off - c.base)
	n = copy(p, c.cache[needSkip:c.size])
	if (needSkip + n) == c.size {
		fixedBytePool.Put(c.cache)
		c.cache = nil
	}

	if n != wantN {
		err = io.ErrUnexpectedEOF
	}

	return
}

// NewChunkAlignedReaderAt returns a ReaderAt wrapper that is backed
// by a ReaderAt r of size totalSize where the wrapper guarantees that
// all ReadAt calls are aligned to chunkSize boundaries and of size
// chunkSize (except for the final chunk, which may be shorter).
//
// A chunk-aligned reader is good for caching, letting upper layers have
// any access pattern, but guarantees that the wrapped ReaderAt sees
// only nicely-cacheable access patterns & sizes.
func NewChunkAlignedReaderAt(r SizeReaderAt, chunkSize int) (SizeReaderAt, error) {
	if chunkSize > chunkSizeLimit {
		// NOTE(wenjianhn): Do you really need a chunk that is such large?
		return &multi{}, errors.New("chunkaligned: chunk size limit exceeded")
	}

	totalSize := r.Size()

	partN := int(math.Ceil(float64(totalSize) / float64(chunkSize)))
	m := &multi{
		parts: make([]offsetAndSource, partN),
		size:  totalSize,
	}

	left := m.Size()
	var offset int64
	for i, _ := range m.parts {
		partSize := chunkSize
		if left < int64(chunkSize) {
			// the final chunk is shorter
			partSize = int(left)
		}
		f := &chunkReaderAt{
			size:  partSize,
			base:  offset,
			cache: nil,
			r:     r,
		}
		m.parts[i] = offsetAndSource{offset,
			io.NewSectionReader(f, offset, f.Size())}
		offset += f.Size()
		left -= f.Size()
	}
	return m, nil
}
