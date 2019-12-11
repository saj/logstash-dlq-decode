package dlq

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	v1BlockSize = 32 * 1024
)

const (
	// https://github.com/elastic/logstash/blob/7f5aa186c1e395bfb8eda8b1c415502c9baa8cb5/logstash-core/src/main/java/org/logstash/common/io/RecordType.java

	v1RecordTypeComplete = 'c'
	v1RecordTypeStart    = 's'
	v1RecordTypeMiddle   = 'm'
	v1RecordTypeEnd      = 'e'
)

type v1RecordHeader struct {
	Type       byte
	RecordSize int32
	EventSize  int32
	Checksum   int32
}

type v1Decoder struct {
	r     io.Reader
	err   error
	eof   bool
	b     *v1Block
	event []byte
}

func newV1Decoder(r io.Reader) *v1Decoder {
	return &v1Decoder{r: r}
}

func (d *v1Decoder) Scan() bool {
	if d.eof {
		return false
	}

	if d.b == nil {
		if !d.scanBlock() {
			return false
		}
	}

	// The initial capacity of this slice is not significant.  Use block size in
	// an attempt to avoid allocs:  we assume most events fit within one block.
	d.event = make([]byte, 0, v1BlockSize)

	var complete bool
	for !complete {
		var (
			data []byte
			err  error
		)
		data, complete, err = d.b.NextRecord()
		d.event = append(d.event, data...)
		if !complete && err == io.EOF {
			if !d.scanBlock() {
				return false
			}
			continue
		}
		if err != nil {
			d.err = err
			return false
		}
	}
	return true
}

func (d *v1Decoder) scanBlock() bool {
	var err error
	d.b, err = readBlock(d.r)
	if err == io.EOF {
		d.eof = true
		if d.b == nil {
			return false
		}
	} else if err != nil {
		d.err = err
		return false
	}
	return true
}

func (d *v1Decoder) Err() error {
	return d.err
}

func (d *v1Decoder) RawEvent() []byte {
	return d.event
}

type v1Block struct {
	b      *bytes.Reader
	offset int
}

func readBlock(r io.Reader) (*v1Block, error) {
	var (
		b   = make([]byte, v1BlockSize)
		z   int
		err error
	)
	for err == nil && z < v1BlockSize {
		var n int
		n, err = r.Read(b[z:])
		z += n
	}
	if z == 0 {
		return nil, err
	}
	return &v1Block{b: bytes.NewReader(b)}, err
}

func (b *v1Block) NextRecord() (data []byte, complete bool, err error) {
	var hdr v1RecordHeader
	err = binary.Read(b.b, binary.BigEndian, &hdr)
	if err != nil {
		return
	}

	switch t := hdr.Type; t {
	case v1RecordTypeStart:
	case v1RecordTypeMiddle:
	case v1RecordTypeEnd:
		fallthrough
	case v1RecordTypeComplete:
		complete = true
	default:
		err = fmt.Errorf("unknown record type: %x", t)
		return
	}

	data = make([]byte, hdr.RecordSize)
	_, err = b.b.Read(data)
	return
}
