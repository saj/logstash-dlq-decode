package dlq

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"time"

	"github.com/ugorji/go/codec"
)

type Event struct {
	Timestamp  time.Time     `json:"timestamp"`
	Event      []interface{} `json:"event"`
	PluginType string        `json:"plugin_type"`
	PluginID   string        `json:"plugin_id"`
	Reason     string        `json:"reason"`
}

var (
	codecHandle = &codec.CborHandle{}
)

func init() {
	// DecodeOptions is an embedded field; it is exported via BasicHandle.
	// BasicHandle is marked as deprecated in the codec package documentation.
	// Promoted fields cannot be used as field names in composite literals.
	// This is an odd situation.  As a workaround, we make an assignment
	// separately, here, using a field selector.
	codecHandle.DecodeOptions = codec.DecodeOptions{
		MapType: reflect.TypeOf(map[string]interface{}{}),
	}
}

func (e *Event) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	{
		b, err := readLengthPrefixedBytes(buf)
		if err != nil {
			return err
		}
		t, err := time.Parse(time.RFC3339Nano, string(b))
		if err != nil {
			return err
		}
		e.Timestamp = t
	}
	{
		b, err := readLengthPrefixedBytes(buf)
		if err != nil {
			return err
		}
		dec := codec.NewDecoderBytes(b, codecHandle)
		if err := dec.Decode(&(e.Event)); err != nil {
			return err
		}
	}
	{
		b, err := readLengthPrefixedBytes(buf)
		if err != nil {
			return err
		}
		e.PluginType = string(b)
	}
	{
		b, err := readLengthPrefixedBytes(buf)
		if err != nil {
			return err
		}
		e.PluginID = string(b)
	}
	{
		b, err := readLengthPrefixedBytes(buf)
		if err != nil {
			return err
		}
		e.Reason = string(b)
	}
	return nil
}

func readLengthPrefixedBytes(r io.Reader) ([]byte, error) {
	var l int32
	err := binary.Read(r, binary.BigEndian, &l)
	if err != nil {
		return nil, err
	}

	var (
		b = make([]byte, l)
		z int32
	)
	for err == nil && z < l {
		var n int
		n, err = r.Read(b[z:])
		z += int32(n)
	}
	return b, err
}
