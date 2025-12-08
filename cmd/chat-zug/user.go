package main

import "net"

type User struct {
	Name    string
	Addr    string
	MsgChan chan string

	conn   net.Conn
	server *Server
}

func NewUser(conn net.Conn, s *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:    userAddr,
		Addr:    userAddr,
		MsgChan: make(chan string),
		conn:    conn,
		server:  s,
	}

	go user.ListenMsg()

	return user
}

func (user *User) Online() {
	user.server.mapLock.Lock()
	user.server.UserMap[user.Name] = user
	user.server.mapLock.Unlock()

	user.server.Broadcast(user, "connected")
}

func (user *User) Offline() {
	user.server.mapLock.Lock()
	delete(user.server.UserMap, user.Name)
	user.server.mapLock.Unlock()

	user.server.Broadcast(user, "offline")
}

func (user *User) OnMessage(msg string) {
	user.server.Broadcast(user, msg)
}

func (user *User) ListenMsg() {
	for {
		msg := <-user.MsgChan

		user.conn.Write([]byte(msg + "\n"))
	}
}
