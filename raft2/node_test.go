package raft

import (
	"reflect"
	"sort"
	"testing"
)

const (
	a = iota + 1
	b
	c
)

type byTo []Message

func (ms byTo) Len() int           { return len(ms) }
func (ms byTo) Less(i, j int) bool { return ms[i].To < ms[j].To }
func (ms byTo) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }

func TestStateIsLeader(t *testing.T) {
	tests := []struct {
		s State
		w bool
	}{
		{State{Id: 1, Lead: 1}, true},
		{State{Id: 2, Lead: 2}, true},
		{State{Id: 1, Lead: 2}, false},
		{State{Id: 2, Lead: 1}, false},
	}

	for i, tt := range tests {
		if g := tt.s.IsLeader(); g != tt.w {
			t.Errorf("#%d: IsLeader = %v, want %v", i, g, tt.w)
		}
	}
}

func TestStateIsCandidate(t *testing.T) {
	tests := []struct {
		s State
		w bool
	}{
		{State{Id: 1, Vote: 1, Lead: 0}, true},
		{State{Id: 2, Vote: 2, Lead: 0}, true},
		{State{Id: 1, Vote: 2, Lead: 0}, false},
		{State{Id: 2, Vote: 1, Lead: 0}, false},
		{State{Id: 2, Vote: 2, Lead: 2}, false},
	}

	for i, tt := range tests {
		if g := tt.s.IsCandidate(); g != tt.w {
			t.Fatalf("#%d: IsCandidate != %v, want %v", i, g, tt.w)
		}
	}
}

func TestCampaign(t *testing.T) {
	tests := []struct {
		replies []Message
		w       bool
	}{
		{
			[]Message{
				{State: State{Id: b, Vote: b, Term: 1}},
				{State: State{Id: c, Vote: c, Term: 1}},
			},
			false,
		},
		{
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: c, Vote: b, Term: 1}},
			},
			true,
		},
		{
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: c, Vote: a, Term: 1}},
			},
			true,
		},

		// votes are all in previous term
		{
			[]Message{
				{State: State{Id: b, Vote: a, Term: 0}},
				{State: State{Id: c, Vote: a, Term: 0}},
			},
			false,
		},
	}

	for i, tt := range tests {
		n := New(a, b, c)
		n.Campaign()
		for _, m := range tt.replies {
			n.Step(m)
		}
		if g := n.s.IsLeader(); g != tt.w {
			t.Errorf("#%d: IsLeader = %v, want %v", i, g, tt.w)
		}
	}
}

func TestCampaignMessages(t *testing.T) {
	n := New(a, b, c)
	n.log = append(n.log, Entry{1, 1})
	n.Campaign()
	g := n.ReadMessages()
	w := []Message{
		{To: b, State: *n.s, Mark: Mark{1, 1}},
		{To: c, State: *n.s, Mark: Mark{1, 1}},
	}
	sort.Sort(byTo(g))
	if !reflect.DeepEqual(g, w) {
		t.Errorf("\ng  = %+v\nwant %+v", g, w)
	}
}
