package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	server *Server
}

//创建用户
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	//启动监听当前user channel 的goroutine
	go user.ListenMessage()
	return user
}

//用户上线
func (this *User) Online() {

	//用户上线，将用户添加到online map
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()
	this.server.BroadCast(this, "已上线\n")
}

//用户下线
func (this *User) Offline() {

	//用户下线，将用户从online map删除
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	//广播当前用户下线消息
	this.server.BroadCast(this, "下线\n")
}

func (this *User) SendMsg(msg string) {
	this.conn.Write([]byte(msg))
}

//用户处理消息
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前全部用户
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.SendMsg(onlineMsg)
		}
		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]

		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("用户名已被占用\n")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()
			this.Name = newName
			this.SendMsg("用户名修改成功，新的用户名:" + newName + "\n")
		}

	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式: to|用户名|消息
		//获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.SendMsg("消息格式不正确，请使用\"to|用户名|消息\"格式\n")
			return
		}
		//根据用户名获取User对象
		remoteUser, ok := this.server.OnlineMap[remoteName]
		if !ok {
			this.SendMsg("该用户不存在\n")
			return
		}
		//获取消息内容，通过User对象发送消息
		content := strings.Split(msg, "|")[2]
		if content == "" {
			this.SendMsg("消息为空，请重新输入\n")
			return
		} else {
			remoteUser.SendMsg(this.Name + "对您说:" + content + "\n")
		}
	} else {
		this.server.BroadCast(this, msg)
	}
}

//监听当前User channel的 方法,一旦有消息，就直接发送给对端客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}
