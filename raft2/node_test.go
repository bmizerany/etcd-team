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
			newTestNode(a),
			nil,
			true,
		},
		{
			newTestNode(a, b, c),
			nil,
			false,
		},
		{
			newTestNode(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: b, Term: 1}},
				{State: State{Id: c, Vote: c, Term: 1}},
			},
			false,
		},
		{
			newTestNode(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: c, Vote: b, Term: 1}},
			},
			true,
		},
		{
			newTestNode(a, b, c),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: c, Vote: a, Term: 1}},
			},
			true,
		},
		{
			// duplicate vote
			newTestNode(a, b, c, d, e),
			[]Message{
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: b, Vote: a, Term: 1}},
				{State: State{Id: b, Vote: a, Term: 1}},
			},
			false,
		},

		// votes are all in previous term
		{
			newTestNode(a, b, c),
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
			newTestNode(a, b, c),
			[]Message{
				{To: b, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: c, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
			},
		},
		{
			newTestNode(a, b, c, d, e),
			[]Message{
				{To: b, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: c, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: d, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
				{To: e, State: State{Term: 1, Id: a, Vote: a}, Mark: Mark{1, 1}},
			},
		},
		{
			newTestNode(a),
			nil,
		},
	}
	for i, tt := range tests {
		tt.n.log = append(tt.n.log, Entry{Index: 1, Term: 1})
		tt.n.Campaign()
		g := tt.n.ReadMessages()
		sort.Sort(byTo(g))
		if !reflect.DeepEqual(g, tt.wantSent) {
			t.Errorf("#%d: \ng  = %+v\nwant %+v", i, g, tt.wantSent)
		}
	}
}

func TestNewLeaderSendsInitAppend(t *testing.T) {
	n := newTestNode(a, b, c)
	n.coerceLeader()

	g := n.ReadMessages()
	sort.Sort(byTo(g))
	w := []Message{
		{
			To:      b,
			State:   State{Term: 1, Id: a, Vote: a, Lead: a},
			Mark:    Mark{1, 0},
			Entries: []Entry{{Index: 2, Term: 1}},
		},
		{
			To:      c,
			State:   State{Term: 1, Id: a, Vote: a, Lead: a},
			Mark:    Mark{1, 0},
			Entries: []Entry{{Index: 2, Term: 1}},
		},
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("\ng  = %+v\nwant %+v", g, w)
	}
}

func TestLeaderRecvAppendResponse(t *testing.T) {
	n := newTestNode(a, b, c)
	n.coerceLeader()
	n.ReadMessages() // discard init appends

	n.Step(Message{State: State{Term: 1, Id: b, Lead: a}, Mark: Mark{0, 0}})

	w := []Message{
		{
			To:    b,
			State: State{Term: 1, Id: a, Vote: a, Lead: a},
			Mark:  Mark{0, 0},

			// entries including init peer list and nop added by leader
			Entries: []Entry{
				{Index: 1, Peers: []int64{a, b, c}},
				{Index: 2, Term: 1},
			},
		},
	}
	g := n.ReadMessages()
	sort.Sort(byTo(g))
	if !reflect.DeepEqual(g, w) {
		t.Errorf("\ng  = %+v\nwant %+v", g, w)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		g *Node
		w *Node
		p bool
	}{
		{
			g: New(a, nil),
			w: &Node{
				State: State{Id: a},

				// start the world at Term 0
				log:   []Entry{{}, {Index: 1, Id: a, Term: 0}},
				peers: peers{},
			},
		},
		{
			g: newTestNode(a, b),
			w: &Node{
				State: State{Id: a},
				log:   []Entry{{}, {Index: 1, Peers: []int64{a, b}}},
				peers: peers{b: {}},
			},
		},
	}

	for i, tt := range tests {
		if !reflect.DeepEqual(tt.g, tt.w) {
			t.Errorf("#%d: \ng  = %+v\nwant %+v", i, tt.g, tt.w)
		}
	}
}

func TestCheckLog(t *testing.T) {
	tests := []struct {
		id  int64
		log []Entry
	}{
		{a, []Entry{{Index: 90, Term: 5}, {Index: 90, Term: 5}}},
		{a, []Entry{{Index: 90, Term: 5}, {Index: 99, Term: 5}}},
		{a, []Entry{{Index: 90, Term: 5}, {Index: 91, Term: 4}}},
	}

	for i, tt := range tests {
		func() {
			defer func() {
				e := recover()
				if e == nil {
					t.Errorf("#%d: expected panic", i)
				}
			}()
			New(tt.id, tt.log)
		}()
	}
}

func newTestNode(ids ...int64) *Node {
	return New(ids[0], []Entry{{}, {Index: 1, Peers: ids}})
}

func (n *Node) coerceLeader() {
	n.Campaign()
	n.ReadMessages() // discard vote requests

	n.Step(Message{State: State{Term: 1, Id: b, Vote: a}}) // final vote in quorum
	if !n.IsLeader() {
		panic("expected leader")
	}
}
