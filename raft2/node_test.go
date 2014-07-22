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
	d
	e
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
		n          *Node
		replies    []Message
		wantLeader bool
	}{
		{
			New(a),
			nil,
			true,
		},
		{
			New(a, b, c),
			nil,
			false,
		},
		{
			New(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: b, Term: 1}},
				{State: State{Id: c, Vote: c, Term: 1}},
			},
			false,
		},
		{
			New(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: c, Vote: b, Term: 1}},
			},
			true,
		},
		{
			New(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: c, Vote: a, Term: 1}},
			},
			true,
		},
		{
			// duplicate vote
			New(a, b, c, d, e),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: b, Vote: a, Term: 1}},
			},
			false,
		},

		// votes are all in previous term
		{
			New(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 0}},
				{State: State{Id: c, Vote: a, Term: 0}},
			},
			false,
		},
	}

	for i, tt := range tests {
		tt.n.Campaign()
		if len(tt.n.peers) > 0 && !tt.n.IsCandidate() {
			t.Errorf("#%d: expected candidate", i)
		}
		for _, m := range tt.replies {
			tt.n.Step(m)
		}
		if g := tt.n.IsLeader(); g != tt.wantLeader {
			t.Errorf("#%d: IsLeader = %v, want %v", i, g, tt.wantLeader)
		}
	}
}

func TestCampaignSendsVoteRequests(t *testing.T) {
	tests := []struct {
		n        *Node
		wantSent []Message
	}{
		{
			New(a, b, c),
			[]Message{
				{To: b, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: c, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
			},
		},
		{
			New(a, b, c, d, e),
			[]Message{
				{To: b, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: c, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: d, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: e, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
			},
		},
		{
			New(a),
			nil,
		},
	}
	for i, tt := range tests {
		tt.n.log = append(tt.n.log, Entry{1, 1})
		tt.n.Campaign()
		g := tt.n.ReadMessages()
		sort.Sort(byTo(g))
		if !reflect.DeepEqual(g, tt.wantSent) {
			t.Errorf("#%d: \ng  = %+v\nwant %+v", i, g, tt.wantSent)
		}
	}
}

func TestNewLeaderSendsInitAppend(t *testing.T) {
	n := New(a, b, c)
	n.coerceLeader()

	g := n.ReadMessages()
	sort.Sort(byTo(g))
	w := []Message{
		{
			To:      b,
			State:   State{Term: 1, Id: a, Vote: a, Lead: a},
			Mark:    Mark{0, 0},
			Entries: []Entry{{1, 1}},
		},
		{
			To:      c,
			State:   State{Term: 1, Id: a, Vote: a, Lead: a},
			Mark:    Mark{0, 0},
			Entries: []Entry{{1, 1}},
		},
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("\ng  = %+v\nwant %+v", g, w)
	}
}

func TestLeaderRecvAppendResponse(t *testing.T) {
	n := New(a, b, c)
	n.coerceLeader()
	n.ReadMessages() // discard init appends

	n.Step(Message{State: State{Term: 1, Id: b, Lead: a}, Mark: Mark{0, 0}})

	w := []Message{
		{To: b, State: State{Term: 1, Id: a, Vote: a, Lead: a}, Entries: []Entry{{Term: 1, Index: 1}}, Mark: Mark{0, 0}},
	}
	g := n.ReadMessages()
	sort.Sort(byTo(g))
	if !reflect.DeepEqual(g, w) {
		t.Errorf("\ng  = %+v\nwant %+v", g, w)
	}
}

func (n *Node) coerceLeader() {
	n.Campaign()
	n.ReadMessages() // discard vote requests

	n.Step(Message{State: State{Term: 1, Id: b, Vote: a}}) // final vote in quorum
	if !n.IsLeader() {
		panic("expected leader")
	}
}
