package main

import (
	"net"
	"strings"
)

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

func (user *User) SendMsg(msg string) {
	user.conn.Write([]byte(msg))
}

func (user *User) OnMessage(msg string) {
	if msg == "who" {
		user.server.mapLock.Lock()
		for _, u := range user.server.UserMap {
			iden := "[" + u.Addr + "]" + u.Name + ": online\n"
			user.SendMsg(iden)
		}
		user.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		_, ok := user.server.UserMap[newName]
		if ok {
			user.SendMsg("user name is already in use")
		} else {
			user.server.mapLock.Lock()
			delete(user.server.UserMap, user.Name)
			user.server.UserMap[newName] = user
			user.server.mapLock.Unlock()

			user.Name = newName
			user.SendMsg("user name changed to" + newName + " successfully\n")
		}
	} else {
		user.server.Broadcast(user, msg)
	}
}

func (user *User) ListenMsg() {
	for {
		msg := <-user.MsgChan

		user.conn.Write([]byte(msg + "\n"))
	}
}
