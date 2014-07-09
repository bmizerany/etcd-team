package wal

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"testing"

	"code.google.com/p/gogoprotobuf/proto"
)

func TestDecode(t *testing.T) {
	tests := []*Record{
		record(1, 7, "a"),
		record(2, 8, "b"),
		record(3, 9, "x"),
	}

	b := new(bytes.Buffer)
	for _, tt := range tests {
		tencode(b, tt)
	}

	d := NewDecoder(b)
	for i, tt := range tests {
		g := new(Record)
		if err := d.Decode(g); err != nil {
			t.Errorf("#%d: error - %s", i, err)
		}
		if !reflect.DeepEqual(g, tt) {
			t.Errorf("got = %+v, want %v", g, tt)
		}
	}

	if err := d.Decode(nil); err != io.EOF {
		t.Errorf("err = %v, want io.EOF", err)
	}
}

func record(index uint64, t int64, s string) *Record {
	return &Record{Index: index, Prev: index - 1, Type: t, Data: []byte(s)}
}

func tencode(w io.Writer, rec *Record) {
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
