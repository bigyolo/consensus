package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"time"
)

func replyClient(context []byte, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("connect error", err)
		return
	}
	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

//节点使用的tcp监听,
func (p *PBFT) tcpListen() {

	listen, err := net.Listen("tcp", p.node.addr)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", p.node.addr)
	defer listen.Close()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}

		b, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		p.HandleRequest(b)
	}

}

//使用tcp发送消息
func tcpDial(context []byte, addr string) {
	r := 30
	time.Sleep(time.Duration(r) * time.Millisecond)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("connect error", err)
		return
	}

	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}

//产生随机值
func randRange(min, max int64) int64 {
	//用于心跳信号的时间
	rand.Seed(time.Now().UnixNano())
	return rand.Int63n(max-min) + min
}
