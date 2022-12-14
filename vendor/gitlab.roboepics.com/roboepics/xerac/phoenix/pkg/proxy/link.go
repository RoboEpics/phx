package proxy

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to read the next pong message from the client.
	pongWait = 120 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type Link interface {
	Writer() chan<- []byte
	Reader() <-chan []byte
	RemoteAddr() string
	Close() error
}

type Dialer func() (Link, error)
type Acceptor func() (Link, string, string, error)

type tcpLink struct {
	conn     net.Conn
	wch, rch chan []byte
	closed   chan struct{}
}

func TCPConn(conn net.Conn) Link {
	tl := &tcpLink{
		conn:   conn,
		wch:    make(chan []byte, 32),
		rch:    make(chan []byte, 32),
		closed: make(chan struct{}),
	}
	go tl.read()
	go tl.write()
	return tl
}

func TCPDialer(ip string, port int) Dialer {
	addr := fmt.Sprintf("%s:%d", ip, port)
	return func() (Link, error) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		tl := &tcpLink{
			conn:   conn,
			wch:    make(chan []byte, 32),
			rch:    make(chan []byte, 32),
			closed: make(chan struct{}),
		}
		go tl.read()
		go tl.write()
		return tl, nil
	}
}

func (tl *tcpLink) read() {
	defer tl.Close()
	defer close(tl.rch)
	const kb = 1024
	buf := make([]byte, 4*kb)
	for {
		n, err := tl.conn.Read(buf)
		if n > 0 {
			// TODO: How about using
			//  sync.Pool?
			cbuf := make([]byte, n)
			copy(cbuf, buf)
			tl.rch <- cbuf
		}
		if err != nil {
			return
		}
	}
}

func (tl *tcpLink) write() {
	defer tl.Close()
	for {
		select {
		case msg, ok := <-tl.wch:
			if !ok {
				return
			}
			_, err := tl.conn.Write(msg)
			time.Sleep(1 * time.Second)
			if err != nil {
				return
			}
		case <-tl.closed:
			return
		}
	}
}

func (tl *tcpLink) Writer() chan<- []byte {
	return tl.wch
}

func (tl *tcpLink) Reader() <-chan []byte {
	return tl.rch
}

func (tl *tcpLink) RemoteAddr() string {
	return tl.conn.RemoteAddr().String()
}

func (tl *tcpLink) Close() error {
	select {
	case <-tl.closed:
	default:
		close(tl.closed)
	}
	return tl.conn.Close()
}

type websocketLink struct {
	ws       *websocket.Conn
	wch, rch chan []byte
	closed   chan struct{}
}

func WebsocketAcceptor(allow func(name, secret string) bool) (http.Handler, Acceptor) {
	const kb = 1024
	// TODO: buffer pool?
	upgrader := websocket.Upgrader{
		ReadBufferSize:  32 * kb,
		WriteBufferSize: 32 * kb,
	}
	type wslink struct {
		conn         *websocket.Conn
		name, secret string
	}
	ch := make(chan wslink, 32)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get("name")
		secret := r.Header.Get("secret")
		if !allow(name, secret) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		ws, err := upgrader.Upgrade(w, r, http.Header{})
		if err != nil {
			return
		}
		ch <- wslink{
			name:   name,
			secret: secret,
			conn:   ws,
		}
	})
	a := func() (Link, string, string, error) {
		conn := <-ch
		wl := &websocketLink{
			ws:     conn.conn,
			wch:    make(chan []byte, 32),
			rch:    make(chan []byte, 32),
			closed: make(chan struct{}),
		}
		go wl.read()
		go wl.write()
		return wl, conn.name, conn.secret, nil
	}
	return h, a
}

func WebsocketDialer(addr, name, secret string) Dialer {
	return func() (Link, error) {
		h := http.Header{}
		h.Set("name", name)
		h.Set("secret", secret)
		conn, _, err := websocket.DefaultDialer.Dial(addr, h)
		if err != nil {
			return nil, err
		}
		wl := &websocketLink{
			ws:     conn,
			wch:    make(chan []byte, 32),
			rch:    make(chan []byte, 32),
			closed: make(chan struct{}),
		}
		go wl.read()
		go wl.write()
		return wl, nil
	}
}

func (wl *websocketLink) read() {
	defer wl.Close()
	defer close(wl.rch)
	for {
		wl.ws.SetReadDeadline(
			time.Now().Add(pongWait))
		wl.ws.SetPongHandler(
			func(string) error {
				wl.ws.SetReadDeadline(
					time.Now().Add(pongWait))
				return nil
			})
		_, msg, err := wl.ws.ReadMessage()
		if err != nil {
			return
		}
		if len(msg) <= 0 {
			continue
		}
		select {
		case wl.rch <- msg:
		case <-wl.closed:
			return
		}
	}
}

func (wl *websocketLink) write() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		wl.Close()
	}()
	for {
		select {
		case <-pingTicker.C:
			if err := wl.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logrus.Errorln("ERR!", err)
				return
			}
		case <-wl.closed:
			logrus.Infoln("CLOSED WEBSOCKET WIRE")
			return
		default:
			select {
			case msg, ok := <-wl.wch:
				if !ok {
					return
				}
				log := logrus.WithFields(logrus.Fields{
					"MSG_SIZE": len(msg),
					"OK":       ok,
					"CH_SIZE":  len(wl.wch),
				})
				log.Traceln("WRITING TO WIRE:\n", string(msg))
				time.Sleep(1 * time.Second)
				err := wl.ws.WriteMessage(websocket.BinaryMessage, msg)
				log.Traceln("DONE    TO WIRE ===================================<<<")
				if err != nil {
					log.Errorln("ERR!", err)
					return
				}
			case <-pingTicker.C:
				if err := wl.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					logrus.Errorln("ERR!", err)
					return
				}
			case <-wl.closed:
				logrus.Infoln("CLOSED WEBSOCKET WIRE")
				return
			}
		}
	}
}

func (wl *websocketLink) Writer() chan<- []byte {
	return wl.wch
}

func (wl *websocketLink) Reader() <-chan []byte {
	return wl.rch
}

func (wl *websocketLink) RemoteAddr() string {
	return wl.ws.RemoteAddr().String()
}

func (wl *websocketLink) Close() error {
	select {
	case <-wl.closed:
	default:
		close(wl.closed)
	}
	return wl.ws.Close()
}
