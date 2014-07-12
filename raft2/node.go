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
	*State

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
		State: &State{Id: id},
		peers: ps,
		log:   []Entry{{}},
	}
	return n
}

func (n *Node) Campaign() {
	// start a new term
	n.Term++
	n.Vote = n.Id

	l := n.log[len(n.log)-1]
	for id := range n.peers {
		n.send(Message{To: id, Mark: Mark{l.Index, l.Term}})
	}
}

func (n *Node) Step(m Message) {
	if _, ok := n.peers[m.Id]; !ok {
		return
	}
	n.peers[m.Id] = m.State

	if n.hasMajority() {
		n.Lead = n.Id
	}
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
	m.State = *n.State
	n.msgs = append(n.msgs, m)
}

func (n *Node) ReadMessages() []Message {
	msgs := n.msgs
	n.msgs = nil
	return msgs
}
