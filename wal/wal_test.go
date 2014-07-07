package wal

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"code.google.com/p/gogoprotobuf/proto"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		t int
		s string
	}{
		{1, "a"},
		{2, "b"},
		{5, "x"},
	}

	b := new(bytes.Buffer)
	for _, tt := range tests {
		tencode(b, tt.t, tt.s)
	}

	d := NewDecoder(b)
	for i, tt := range tests {
		r := new(Record)
		if err := d.Decode(r); err != nil {
			t.Errorf("#%d: error - %s", i, err)
		}
		if g := int(r.Type); g != tt.t {
			t.Errorf("#%d: r.Type = %d, want %d", i, g, tt.t)
		}
		if g := string(r.Data); g != tt.s {
			t.Errorf("#%d: r.Data = %q, want %q", i, g, tt.s)
		}
	}

	if err := d.Decode(nil); err != io.EOF {
		t.Errorf("err = %v, want io.EOF", err)
	}
}

func tencode(w io.Writer, t int, s string) {
	rec := &Record{Type: int64(t), Data: []byte(s)}
	rec.UpdateChecksum()
	b := mustMarshal(rec)
	mustWrite(w, int64(len(b)))
	mustWrite(w, b)
}

func mustWrite(w io.Writer, v interface{}) {
	err := binary.Write(w, binary.BigEndian, v)
	if err != nil {
		panic(err)
	}
}

func mustMarshal(v proto.Message) []byte {
	data, err := proto.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
