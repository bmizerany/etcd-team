package raft

type State struct {
	Id   int64
	Vote int64
	Lead int64

	Term int64
}

func (s *State) IsLeader() bool {
	return s.Lead == s.Id
}

func (s *State) IsCandidate() bool {
	return !s.IsLeader() && s.Vote == s.Id
}

type Message struct {
	State

	To      int64
	Mark    Mark
	Entries []Entry

	// Quit signals the reciever to quit participating if they are know by
	// sender to have been removed.
	Quit bool
}

type Mark struct {
	Index int64
	Term  int64
}

type Entry struct {
	Index int64
	Term  int64
}

type peers map[int64]State

type Node struct {
	State

	// the state of all peers, not including self
	peers peers

	log []Entry

	msgs []Message
}

func New(id int64, peerids ...int64) *Node {
	if id == 0 {
		panic("raft: id cannot be 0")
	}
	ps := make(peers)
	for _, pid := range peerids {
		ps[pid] = State{}
	}
	n := &Node{
		State: State{Id: id},
		peers: ps,
		log:   []Entry{{}},
	}
	return n
}

func (n *Node) Campaign() {
	n.Term++
	n.Vote = n.Id

	if len(n.peers) == 0 {
		// we have no peers so we automaticly win the election
		n.Lead = n.Id
	} else {
		l := n.log[len(n.log)-1]
		for id := range n.peers {
			n.send(Message{To: id, Mark: Mark{l.Index, l.Term}})
		}
	}
}

// Step advances nodes state based on m.
func (n *Node) Step(m Message) {
	n.peers[m.Id] = m.State

	switch {
	case n.IsLeader():
		// ensure they acknowledge us as leader
		if m.Lead == n.Id {
			n.handleAppendResponse(m)
		}
	case n.IsCandidate():
		if m.Vote == n.Id {
			n.handleVoteResponse(m)
		}
	}
}

func (n *Node) handleVoteResponse(m Message) {
	if n.hasMajority() {
		n.becomeLeader()
	}
}

func (n *Node) handleAppendResponse(m Message) {
	l := n.log[len(n.log)-1]
	if m.Mark.Index < l.Index {
		e := n.log[m.Mark.Index]
		r := n.log[m.Mark.Index+1:]
		m := Message{
			To:      m.Id,
			Mark:    Mark{e.Index, e.Term},
			Entries: r,
		}
		n.send(m)
	}
}

func (n *Node) becomeLeader() {
	n.Lead = n.Id

	// append an nop to cause everyone to advance quickly
	e := n.log[len(n.log)-1]
	l := n.append(nil)
	for id := range n.peers {
		m := Message{
			To:      id,
			Mark:    Mark{e.Index, e.Term},
			Entries: []Entry{l},
		}
		n.send(m)
	}
}

func (n *Node) append(data []byte) Entry {
	e := n.log[len(n.log)-1]
	// TODO(bmizerany): add data field
	e = Entry{Index: e.Index + 1, Term: n.Term}
	n.log = append(n.log, e)
	return e
}

func (n *Node) hasMajority() bool {
	g := 0
	for _, s := range n.peers {
		if s.Term == n.Term && s.Vote == n.Id {
			g++
		}
	}

	k := len(n.peers)
	q := k / 2 // no need to +1 since we don't include ourself in n.peers
	return g >= q
}

// send queues m in the outbox of messages. Messages sent to self are ignored.
func (n *Node) send(m Message) {
	if m.To == n.Id {
		return
	}
	m.State = n.State
	n.msgs = append(n.msgs, m)
}

func (n *Node) ReadMessages() []Message {
	msgs := n.msgs
	n.msgs = nil
	return msgs
}
