package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const reqtimes = 30 //请求20次，取平均值

// 从客户端角度测算耗时如下

var timeStart, timeEnd time.Time

func clientSendMessageAndListen() {
	//开启客户端的本地监听（主要用来接收节点的reply信息）
	go clientTcpListen()
	fmt.Printf("客户端开启监听，地址：%s\n", clientAddr)

	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("|  已进入PBFT测试Demo客户端，请启动全部节点后再发送消息！ :)  |")
	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("请在下方输入要存入节点的信息：")
	//首先通过命令行获取用户输入

	for i := 0; i < reqtimes; i++ {
		// stdReader := bufio.NewReader(os.Stdin)
		// data, err := stdReader.ReadString('\n')
		// if err != nil {
		// 	fmt.Println("Error reading from stdin")
		// 	panic(err)
		// }
		data := "test" + strconv.Itoa(i)
		r := new(Request)
		r.Timestamp = time.Now().UnixNano()
		r.ClientAddr = clientAddr
		r.ViewNum = 0 //主节点 = v % n
		r.Message.ID = getRandom()
		//消息内容就是用户的输入
		r.Message.Content = strings.TrimSpace(data)
		br, err := json.Marshal(r)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(br))
		content := jointMessage(cRequest, br)
		timeStart = time.Now()
		tcpDial(content, nodeTable["N0"]) // 默认N0是主节点
		time.Sleep(20 * time.Second)
	}
	fmt.Println("测算完毕！！！！！")
}

//返回一个十位数的随机数，作为msgid
func getRandom() int {
	x := big.NewInt(10000000000)
	for {
		result, err := rand.Int(rand.Reader, x)
		if err != nil {
			log.Panic(err)
		}
		if result.Int64() > 1000000000 {
			return int(result.Int64())
		}
	}
}

//客户端使用的tcp监听
func clientTcpListen() {
	count := 0
	listen, err := net.Listen("tcp", clientAddr)
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
		count++
		if count%nodeCount == 0 {
			timeEnd = time.Now()
			t := timeEnd.Sub(timeStart)

			fmt.Println("client耗时：", t, count, nodeCount)
			// 写文件记录时间
			w.WriteString(t.String()[0:len(t.String())-2] + "\n")
			w.Flush()

		}
		fmt.Println(string(b), count)
	}

}
