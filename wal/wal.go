package wal

import (
	"encoding/binary"
	"errors"
	"io"

	"code.google.com/p/gogoprotobuf/proto"
)

var ErrInvalidChecksum = errors.New("wal: invalid checksum")

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r}
}

func (d *Decoder) Decode(rec *Record) error {
	// TODO(bmizerany): reuse cached buf?
	b := make([]byte, 8)
	var l int64
	if err := binary.Read(d.r, binary.BigEndian, &l); err != nil {
		return err
	}

	// TODO(bmizerany): reuse cached buf?
	b = make([]byte, int64(l))
	if _, err := io.ReadFull(d.r, b); err != nil {
		return err
	}
	if err := proto.Unmarshal(b, rec); err != nil {
		return err
	}
	if !rec.IsValid() {
		return ErrInvalidChecksum
	}
	return nil
}
