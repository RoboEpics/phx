package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.roboepics.com/roboepics/xerac/phoenix/pkg/util"
)

type Node struct {
	Addr                 string
	MinConns             int
	DialersCount         int
	Key                  []byte
	DisableIncomingConns bool

	peers      map[string]*peer
	listeners  map[net.Listener]struct{}
	usedNonces map[string]struct{}

	mu     sync.Mutex
	closed chan struct{}
}

type IPPort struct {
	IP   string
	Port int
}

func (n *Node) ListenAndServe() error {
	n.init()

	addr := n.Addr
	if addr == "" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	n.mu.Lock()
	n.listeners[ln] = struct{}{}
	n.mu.Unlock()

	return n.Serve(ln)
}

func (n *Node) Serve(l net.Listener) error {
	log := logrus.WithFields(logrus.Fields{})
	n.init()
	log.Infoln("Start serving", l.Addr())
	handler, acceptor := WebsocketAcceptor(func(name, secret string) bool {
		n.mu.Lock()
		defer n.mu.Unlock()
		if p, exists := n.peers[name]; exists {
			return p.secret == secret
		}
		return true
	})
	go func() {
		defer func() {
			log.Warnln("Closing", l.Addr(), "listener")
			l.Close()
		}()
		for {
			ln, name, secret, err := acceptor()
			if err != nil {
				log.Warnln("Accept error", err)
				return
			}
			log := log.WithFields(logrus.Fields{
				"remote_address": ln.RemoteAddr(),
				"name":           name,
			})
			log.Debugln("Accepted")

			n.mu.Lock()
			p, exists := n.peers[name]
			if !exists {
				log.Debugln("Peer not exists; creating one.")
				p = &peer{
					name:        name,
					secret:      secret,
					receive:     n.receive,
					links:       make(map[*linkState]struct{}),
					dialerConns: n.DialersCount,
					minConns:    n.MinConns,
				}
				n.peers[name] = p
			}
			n.mu.Unlock()

			p.mu.Lock()
			lnS := &linkState{
				link: &ln,
				peer: p,
			}
			if p.secret != secret {
				log.Debugln("Invalid secret, dropping link.")
				ln.Close()
				p.mu.Unlock()
			} else {
				p.links[lnS] = struct{}{}
				p.mu.Unlock()
				log.Debugln("Reading and Broadcasting new link")
				go lnS.read(n.receive)
				p.pubsub.Broadcast(lnS)
			}
		}
	}()
	return http.Serve(l, handler)
}

func (n *Node) Connect(name string, d Dialer) error {
	n.init()
	n.mu.Lock()
	defer n.mu.Unlock()
	p := &peer{
		name:        name,
		dialer:      d,
		receive:     n.receive,
		dialerConns: n.DialersCount,
		minConns:    n.MinConns,
		links:       make(map[*linkState]struct{}),
	}
	n.peers[name] = p
	p.checkupLinkCount()
	go n.checkupTimer(p, 5*time.Second)
	return nil
}

func (n *Node) checkupTimer(p *peer, d time.Duration) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p.checkupLinkCount()
		case <-n.closed:
			return
		}
	}
}

func (n *Node) ListenProxy(hops []string, local, remote IPPort) error {
	return n.ListenProxyWithKey(hops, n.Key, local, remote)
}

func (n *Node) ListenProxyWithKey(hops []string, key []byte, local, remote IPPort) error {
	n.init()
	if len(hops) <= 0 {
		return errors.New("empty hops")
	}

	laddr := fmt.Sprintf("%s:%d", local.IP, local.Port)
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	n.mu.Lock()
	n.listeners[l] = struct{}{}
	n.mu.Unlock()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		ln := TCPConn(conn)
		lnS := linkState{
			link:     &ln,
			terminal: true,
		}

		// impersonate terminal link and send connect msg.
		connect := payloadConnect{}
		connect.Hops = hops
		connect.Target.IP = remote.IP
		connect.Target.Port = remote.Port
		if len(key) > 0 {
			connect.diceAndSign(key)
		}
		msg := encodeMsg(magicConnect, connect)
		f := frame{
			body:    msg,
			ln:      &lnS,
			created: true,
		}
		n.receive(f)
	}
}

func (n *Node) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for l := range n.listeners {
		l.Close()
	}
	close(n.closed)
}

func (n *Node) receive(f frame) {
	if f.ln == nil {
		return
	}

	check := func() {
		switch f.magic() {
		case magicData:
			// skip checkup: nop
		default:
			if f.ln != nil && f.ln.peer != nil {
				f.ln.peer.checkupLinkCount()
			}
		}
	}

	// check link count after and before
	//  processing control frames.
	check()
	defer check()

	if f.ln.terminal && !f.created {
		n.receiveRawFrame(f)
		return
	}

	if fn, ok := map[byte]func(frame){
		magicConnect: n.receiveConnectFrame,
		magicAck:     n.receiveAckFrame,
		magicError:   n.receiveCloseErrorFrame,
		magicClose:   n.receiveCloseErrorFrame,
		magicData:    n.receiveDataFrame,
	}[f.magic()]; ok {
		fn(f)
	} else {
		// Drop frames with unknown magic
		_ = 0
	}
}

func (n *Node) receiveRawFrame(f frame) {
	if f.ln == nil || f.ln.attachedTo == nil {
		return
	}
	msg := encodeMsg(magicData, payloadData{Data: f.body})
	n.write(f.ln.attachedTo, msg)
}

func (n *Node) receiveConnectFrame(f frame) {
	log := logrus.WithFields(logrus.Fields{
		"frame": "connect",
	})

	if n.DisableIncomingConns && !f.created {
		return
	}

	var connect payloadConnect
	if err := f.unmarshal(&connect); err != nil {
		log.Warnln("invalid connect frame, dropping:", err)
		return
	}
	log = log.WithFields(logrus.Fields{
		"hops":          connect.Hops,
		"nonce":         connect.Nonce,
		"signature_len": len(connect.Signature),
	})

	if len(connect.Hops) <= 0 {
		if len(n.Key) > 0 && !connect.valid(n.Key) {
			log.Warnln("invalid signature. send back error.")
			msg := encodeMsg(magicError,
				payloadError{Reason: "invalid signature"})
			n.write(f.ln, msg)
			return
		}

		n.mu.Lock()
		if _, exists := n.usedNonces[connect.Nonce]; exists {
			log.Warnln("duplicated nonce. send back error.")
			n.mu.Unlock()
			msg := encodeMsg(magicError,
				payloadError{Reason: "invalid signature"})
			n.write(f.ln, msg)
			return
		} else {
			n.usedNonces[connect.Nonce] = struct{}{}
			n.mu.Unlock()
		}

		ip, port := connect.Target.IP, connect.Target.Port
		d := TCPDialer(ip, port)
		if d == nil {
			return
		}
		ln, err := d()
		if err != nil {
			log.Warnln("cannot dial terminal TCP:", err)
			msg := encodeMsg(magicError,
				payloadError{Reason: "cannot connect: " + err.Error()})
			n.write(f.ln, msg)
			return
		}

		log.Debugln("Registering terminal link")
		lnS := &linkState{
			link:       &ln,
			busy:       true,
			terminal:   true,
			attachedTo: f.ln,
		}

		f.ln.Lock()
		f.ln.attachedTo = lnS
		f.ln.busy = true
		f.ln.waitForAck = false
		f.ln.Unlock()

		// send back ACK
		msg := encodeMsg(magicAck, nil)
		n.write(f.ln, msg)

		// start reading
		go lnS.read(n.receive)

		return
	}

	next, rest := connect.Hops[0], connect.Hops[1:]

	n.mu.Lock()
	nextPeer, found := n.peers[next]
	n.mu.Unlock()

	if !found {
		if !f.ln.terminal {
			// send error: next hop not found
			log.Warnln("next hop not found")
			msg := encodeMsg(magicError,
				payloadError{Reason: "next hop not found: " + next})
			n.write(f.ln, msg)
		} else {
			n.closeTermLink(f.ln)
		}
		return
	}

	log.Debugln("Demanding free link to next hop")
	timeout := 5 * time.Second
	freeLn := nextPeer.freeLink(timeout)
	if freeLn == nil {
		if !f.ln.terminal {
			log.WithField("timeout", timeout).
				Warnln("not enough link to next hop")
			msg := encodeMsg(magicError,
				payloadError{Reason: "not enough link to: " + next})
			n.write(f.ln, msg)
		} else {
			n.closeTermLink(f.ln)
		}
		return
	}

	freeLn.Lock()
	freeLn.busy = true
	freeLn.waitForAck = true
	freeLn.attachedTo = f.ln
	freeLn.Unlock()

	f.ln.Lock()
	f.ln.busy = true
	f.ln.waitForAck = true
	f.ln.attachedTo = freeLn
	f.ln.Unlock()

	connect.Hops = rest
	log.Debugln("Forwarding to next hop")
	msg := encodeMsg(magicConnect, connect)
	n.write(freeLn, msg)
}

func (n *Node) receiveAckFrame(f frame) {
	log := logrus.WithFields(logrus.Fields{
		"frame": "ack",
	})
	log.Debugln("Recieved Ack frame")

	var (
		st  = f.ln
		st2 *linkState
	)
	if st == nil {
		return
	}

	st.Lock()
	st2 = st.attachedTo
	st.waitForAck = false
	st.Unlock()

	if st2 != nil {
		st2.Lock()
		st2.waitForAck = false
		term := st2.terminal
		st2.Unlock()

		if term {
			// start reading
			go st2.read(n.receive)
		} else {
			log.Debugln("Forwarding Ack frame")
			msg := encodeMsg(magicAck, nil)
			n.write(st2, msg)
		}
	}
}

func (n *Node) receiveCloseErrorFrame(f frame) {
	var (
		st  = f.ln
		st2 *linkState
	)
	if st == nil {
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"remote": st.RemoteAddr(),
	})
	switch f.magic() {
	case magicClose:
		log = log.WithField("frame", "close")
		log.Infoln("Recieved close frame")
	case magicError:
		log = log.WithField("frame", "error")
		log.Warnln("Recieved error frame")
	}

	log.Debugln("Reset ingress link")
	// reset ingress link
	st.Lock()
	st2 = st.attachedTo
	st.attachedTo = nil
	st.Unlock()
	st.peer.releaseLink(st)

	if st2 != nil {
		log = log.WithField("remote2", st2.RemoteAddr())
		if st2.terminal {
			log.Debugln("Flush and Close")
			// n.closeTermLink(st2)
			n.flushAndClose(st2)
		} else {
			log.Debugln("Forward close/error frame")
			// forward message:
			n.write(st2, f.body)

			st2.Lock()
			st2.attachedTo = nil
			st2.Unlock()

			log.Debugln("Flush and Close")
			// flush and close
			n.flushAndClose(st2)
		}
	}
}

func (n *Node) receiveDataFrame(f frame) {
	st := f.ln
	if st == nil {
		return
	}

	if !st.busy || st.waitForAck || st.attachedTo == nil {
		// Drop frame
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"log_uid": rand.Int(),
		"frame":   "data",
		"remote":  st.RemoteAddr(),
	})

	if st.attachedTo.terminal {
		// extract payload from data frame.
		var data payloadData
		if err := f.unmarshal(&data); err != nil {
			log.Errorln("Error unmarshal data", err)
			// Drop corrupted frames.
			// TODO: we should not do this!
			//  we must close connection.
			return
		}
		n.write(st.attachedTo, data.Data)
	} else {
		n.write(st.attachedTo, f.body)
	}
}

func (*Node) write(ln *linkState, msg []byte) error {
	if ln == nil {
		return nil
	}
	if ln.closeAfterFlush {
		msg := "unable to write: link wait to close after flush"
		return errors.New(msg)
	}
	wch := (*ln.link).Writer()
	wch <- msg
	return nil
}

func (*Node) flushAndClose(ln *linkState) error {
	if ln == nil {
		return nil
	}
	ln.closeAfterFlush = true
	wch := (*ln.link).Writer()
	close(wch)
	return nil
}

func (n *Node) closeTermLink(ln *linkState) {
	if !ln.terminal {
		return
	}
	ln.Lock()
	if ln := ln.link; ln != nil {
		(*ln).Close()
	}
	ln.attachedTo = nil
	ln.Unlock()
}

func (n *Node) init() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.listeners == nil {
		n.listeners = make(map[net.Listener]struct{})
	}
	if n.peers == nil {
		n.peers = make(map[string]*peer)
	}
	if n.usedNonces == nil {
		n.usedNonces = make(map[string]struct{})
	}
	if n.closed == nil {
		n.closed = make(chan struct{})
	}
}

type frame struct {
	body    []byte
	ln      *linkState
	created bool
}

func (f frame) magic() byte     { return f.body[0] }
func (f frame) payload() []byte { return f.body[1:] }
func (f frame) unmarshal(into any) error {
	return json.Unmarshal(f.payload(), into)
}

type linkState struct {
	link *Link
	peer *peer

	closeAfterFlush bool
	busy            bool
	waitForAck      bool
	terminal        bool
	attachedTo      *linkState

	reading bool
	mu      sync.Mutex
}

func (ls *linkState) Lock()   { ls.mu.Lock() }
func (ls *linkState) Unlock() { ls.mu.Unlock() }
func (ls *linkState) RemoteAddr() string {
	if ls.link != nil {
		return (*ls.link).RemoteAddr()
	}
	return ""
}

func (ls *linkState) read(receive func(frame)) error {
	if ls.reading {
		panic("already reading link")
	}
	ls.reading = true

	rch := (*ls.link).Reader()
	for msg := range rch {
		receive(frame{
			body: msg,
			ln:   ls,
		})
	}
	errmsg := encodeMsg(magicClose, nil)
	receive(frame{
		body:    errmsg,
		ln:      ls,
		created: true,
	})
	return nil
}

type peer struct {
	name        string
	secret      string
	dialer      Dialer
	receive     func(frame)
	dialerConns int
	minConns    int

	links  map[*linkState]struct{}
	pubsub util.Pubsub[*linkState]
	mu     sync.Mutex
}

func (p *peer) checkupLinkCount() (n int, recents chan *linkState) {
	if p.dialer == nil {
		return 0, nil
	}
	log := logrus.WithFields(logrus.Fields{
		"min_conns": p.minConns,
	})

	p.mu.Lock()
	c := 0
	for ln := range p.links {
		ln.Lock()
		if !ln.busy {
			c++
		}
		ln.Unlock()
	}
	p.mu.Unlock()
	log = log.WithField("conns", c)
	log.Debugln(c, "free connections found")

	if diff := p.minConns - c; diff > 0 {
		log.Debugln("running", diff, "dialers")
		recents = make(chan *linkState, diff)
		for i := 0; i < diff; i++ {
			go func() {
				recents <- p.dial()
			}()
		}
		return diff, recents
	}
	return 0, recents
}

func (p *peer) freeLink(timeout time.Duration) *linkState {
	log := logrus.WithFields(logrus.Fields{})

	id, recents := p.pubsub.RegisterN(32)
	defer p.pubsub.Close(id)

	// check all links:
	for ln := range p.links {
		if ln == nil {
			continue
		}
		flag := false
		ln.Lock()
		if !ln.busy {
			ln.busy = true
			flag = true
		}
		ln.Unlock()
		if flag {
			return ln
		}
	}
	// not found, wait for new link:
	log.Debugln("No free link found. waiting for new one.")
	for {
		select {
		case ln, ok := <-recents:
			if !ok {
				return nil
			}
			if ln == nil {
				continue
			}
			// race for getting link
			flag := false
			ln.Lock()
			if !ln.busy {
				ln.busy = true
				flag = true
			}
			ln.Unlock()
			if flag {
				return ln
			}
		case <-time.After(timeout):
			log.Warnln("No free link found after", timeout)
			return nil
		}
	}
}

func (p *peer) releaseLink(ls *linkState) bool {
	log := logrus.WithFields(logrus.Fields{})

	ls.Lock()
	ls.attachedTo = nil
	ls.Unlock()

	// non-terminal links
	if p != nil {
		p.mu.Lock()
		log.Debugln("releasing link", ls)
		delete(p.links, ls)
		p.mu.Unlock()
	}

	if ls.link != nil {
		(*ls.link).Close()
	}
	return false
}

func (p *peer) dial() *linkState {
	log := logrus.WithFields(logrus.Fields{})
	if p.dialer == nil {
		return nil
	}

	ln, err := p.dialer()
	if err != nil {
		log.Errorln("dial error:", err)
		if ln != nil {
			ln.Close()
		}
		return nil
	}

	lnS := &linkState{
		peer: p,
		link: &ln,
	}

	p.mu.Lock()
	log.Infoln("registering new link:", lnS.RemoteAddr())
	p.links[lnS] = struct{}{}
	p.mu.Unlock()
	p.pubsub.Broadcast(lnS)
	go p.read(lnS)

	return lnS
}

func (p *peer) read(ln *linkState) error {
	return ln.read(p.receive)
}
