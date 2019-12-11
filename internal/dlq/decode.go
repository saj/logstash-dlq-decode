package dlq

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
)

const (
	versionSize = 1
)

type versionDecoder interface {
	Scan() bool
	Err() error
	RawEvent() []byte
}

type versionDecoderBuilder func(io.Reader) versionDecoder

var versionDecoders = map[int]versionDecoderBuilder{
	1: func(r io.Reader) versionDecoder { return newV1Decoder(r) },
}

type Decoder struct {
	r   io.Reader
	vd  versionDecoder
	err error
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (d *Decoder) Scan() bool {
	if d.err != nil {
		return false
	}

	if d.vd == nil {
		v, err := readVersion(d.r)
		if err != nil {
			d.err = err
			return false
		}
		vdb, ok := versionDecoders[v]
		if !ok {
			d.err = fmt.Errorf("no decoder for version %d", v)
			return false
		}
		d.vd = vdb(d.r)
	}
	return d.vd.Scan()
}

func (d *Decoder) Err() error {
	if d.err != nil {
		return d.err
	}
	return d.vd.Err()
}

func (d *Decoder) Event() (*Event, error) {
	e := &Event{}
	err := e.UnmarshalBinary(d.vd.RawEvent())
	return e, err
}

func readVersion(r io.Reader) (int, error) {
	vb, err := ioutil.ReadAll(io.LimitReader(r, versionSize))
	if err != nil {
		return -1, fmt.Errorf("version read: %v", err)
	}
	v, err := strconv.Atoi(string(vb))
	if err != nil {
		return -1, fmt.Errorf("version parse: %v", err)
	}
	return v, nil
}
