package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播的channel
	Message chan string
}

//创建一个server 接口

func NewServer(ip string, port int) *Server {
	Server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return Server
}

func (this *Server) ListenMessager() {
	for {
		msg := <-this.Message
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

//广播消息方法
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg
}

func (this *Server) Handler(conn net.Conn) {
	//当前链接业务
	fmt.Printf("链接建立成功")

	user := NewUser(conn, this)

	user.Online()

	//监听用户是否活跃
	isLive := make(chan bool)
	//接受客户端的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {

				fmt.Println("Conn Read err:", err)
				return
			}
			//提取用户消息 去除尾部的换行"/n"
			msg := string(buf[:n-1])

			//用户对msg消息进行处理
			user.DoMessage(msg)

			//重置用户状态，当前用户活跃
			isLive <- true
		}
	}()
	//当前hander阻塞
	for {
		select {
		case <-isLive:
			//当前用户活跃，需要重置计时器
			//什么都不做，为了激活select,更新定时器
		case <-time.After(time.Second * 300):
			//已经超时
			//将当前User强制关闭
			user.SendMsg("你被踢了\n")
			//销毁资源
			close(user.C)
			conn.Close()
			return //runtime.Goexit()
		}
	}
}

func (this *Server) Start() {

	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	defer listener.Close()
	//启动监听message的协程
	go this.ListenMessager()

	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
		//do handeler
		go this.Handler(conn)
	}
}
