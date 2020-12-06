package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var tStart, tEnd time.Time

func clientSendMessage() {
	go clientlisten()
	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("|  已进入Raft测试Demo客户端，请启动全部节点后再发送消息！ :)  |")
	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("请在下方输入要存入节点的信息：")
	//首先通过命令行获取用户输入
	reqtimes := 20
	for i := 0; i < reqtimes; i++ {
		time.Sleep(1 * time.Second)

		data := "test" + strconv.Itoa(i)
		r := new(Logmsg)
		r.Content = data
		r.Timestamp = time.Now().UnixNano()

		br, err := json.Marshal(r)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(br))

		content := jointMessage(Cmessage, br)
		tStart = time.Now()
		tcpDial(content, nodeTable["N0"]) // 向任意一个节点发起request，由节点进行重定向
	}
	select {}
}

func clientlisten() {
	listen, err := net.Listen("tcp", "192.168.242.129:8888")
	if err != nil {
		log.Panic(err)
	}
	defer listen.Close()

	file, err := os.OpenFile("./time.txt", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("打开文件错误= %v \n", err)
		return
	}
	w := bufio.NewWriter(file)
	defer file.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}

		tEnd = time.Now()
		t := tEnd.Sub(tStart)

		fmt.Println("client耗时：", t)
		// 写文件记录时间
		w.WriteString(t.String()[0:len(t.String())-2] + ",")
		w.Flush()

		fmt.Println(string(b))
	}
}
