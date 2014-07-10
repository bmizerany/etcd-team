package wal

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []*Record{
		record(7, 1, "x"),
		record(8, 2, "y"),
		record(9, 3, "z"),
	}

	b := new(bytes.Buffer)
	e := NewEncoder(b)
	for i, tt := range tests {
		if err := e.Encode(tt); err != nil {
			t.Fatalf("#%d: error - %s", i, err)
		}
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
