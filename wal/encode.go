package wal

import (
	"encoding/binary"
	"io"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

func (e *Encoder) Encode(rec *Record) error {
	rec.updateChecksum()

	// TODO(bmizerany): reuse cached buf?
	data, err := rec.Marshal()
	if err != nil {
		return err
	}

	l := int64(len(data))
	if err := binary.Write(e.w, binary.BigEndian, l); err != nil {
		return err
	}

	return binary.Write(e.w, binary.BigEndian, data)
}
