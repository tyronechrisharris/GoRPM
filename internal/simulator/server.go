package simulator

import (
	"log"
	"net"
	"sync"
)

type TCPServer struct {
	address    string
	listener   net.Listener
	clients    map[net.Conn]bool
	clientsMux sync.Mutex
	quit       chan struct{}
}

func NewTCPServer(address string) *TCPServer {
	return &TCPServer{
		address: address,
		clients: make(map[net.Conn]bool),
		quit:    make(chan struct{}),
	}
}

func (s *TCPServer) Start() error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = l

	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					log.Printf("Accept error on %s: %v", s.address, err)
					continue
				}
			}
			s.addClient(conn)
			go s.handleClient(conn)
		}
	}()

	return nil
}

func (s *TCPServer) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	for conn := range s.clients {
		conn.Close()
		delete(s.clients, conn)
	}
}

func (s *TCPServer) addClient(conn net.Conn) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	s.clients[conn] = true
	log.Printf("Client connected: %s", conn.RemoteAddr())
}

func (s *TCPServer) removeClient(conn net.Conn) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	if _, ok := s.clients[conn]; ok {
		delete(s.clients, conn)
		log.Printf("Client disconnected: %s", conn.RemoteAddr())
	}
}

func (s *TCPServer) handleClient(conn net.Conn) {
	defer func() {
		conn.Close()
		s.removeClient(conn)
	}()
	buf := make([]byte, 1024)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return // usually EOF or connection reset
		}
	}
}

func (s *TCPServer) Broadcast(message string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	for conn := range s.clients {
		_, err := conn.Write([]byte(message))
		if err != nil {
			conn.Close()
			delete(s.clients, conn)
		}
	}
}

func (s *TCPServer) ClientCount() int {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	return len(s.clients)
}
