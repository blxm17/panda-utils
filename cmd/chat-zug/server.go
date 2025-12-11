package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Ip       string
	Port     int
	UserMap  map[string]*User
	MsgQueue chan string

	mapLock sync.RWMutex
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:       ip,
		Port:     port,
		UserMap:  make(map[string]*User),
		MsgQueue: make(chan string),
	}

	return server
}

func (s *Server) HandleMsgQueue() {
	for {
		m := <-s.MsgQueue

		s.mapLock.Lock()
		for _, u := range s.UserMap {
			u.MsgChan <- m
		}
		s.mapLock.Unlock()
	}
}

func (s *Server) Broadcast(user *User, m string) {
	sendMsg := "[" + user.Addr + "] " + user.Name + ":  " + m

	s.MsgQueue <- sendMsg
}

func (s *Server) HandleUser(conn net.Conn) {

	defer conn.Close()

	u := NewUser(conn, s)

	u.Online()

	isLive := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				u.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Failed to read message from connection:", err)
				u.Offline()
				return
			}
			m := strings.TrimSpace(string(buf[:n]))
			u.OnMessage(m)

			select {
			case isLive <- true:
			default:
			}
		}
	}()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-isLive:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(10 * time.Second)
		case <-timer.C:
			u.SendMsg("you are inactive for 10 seconds. kickoff")
			close(u.MsgChan)
			u.Offline()
			return
		}
	}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("Open socket listener failed", err)
		return
	}
	defer listener.Close()

	go s.HandleMsgQueue()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Server can't accept connection")
			continue
		}

		go s.HandleUser(conn)
	}
}
