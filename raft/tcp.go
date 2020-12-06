package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
)

//节点使用的tcp监听,必须是一个rpc调用才行，
func (rf *Raft) tcpListen() {
	listen, err := net.Listen("tcp", rf.node.Addr)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", rf.node.Addr)
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
		//fmt.Println(string(b))
		go rf.handleRequest(b)
	}

}

//使用tcp发送消息
func tcpDial(context []byte, addr string) {
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

//向除自己外的其他节点进行广播
func (rf *Raft) broadcast(cmd command, content []byte) {
	for nodeID, addr := range nodeTable {
		if nodeID == rf.node.ID {
			continue
		}
		msg := jointMessage(cmd, content)
		go tcpDial(msg, addr)
	}
}
