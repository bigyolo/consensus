package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/beevik/etree"
)

var nodeCount int

//客户端的监听地址
var clientAddr = "192.168.242.129:8888"

//节点池，主要用来存储监听地址
var nodeTable map[string]string

var tbegin time.Time
var tend time.Time

// 定义节点延迟时间 100ms
var delayTime int

func main() {
	nodeTable = make(map[string]string)
	nodeTable = readnodeTotable("./node.xml")
	nodeCount = len(nodeTable)
	for key, _ := range nodeTable {
		genRsaKeys(key)
	}

	nodeID := os.Args[1]
	if nodeID == "client" {
		clientSendMessageAndListen() //启动客户端程序
	} else if addr, ok := nodeTable[nodeID]; ok {
		p := NewPBFT(nodeID, addr)
		go p.tcpListen() //启动节点
	} else {
		log.Fatal("无此节点编号！")
	}
	if nodeID == "N2" {
		delayTime = 0
	}
	select {}
}

func readnodeTotable(filepath string) map[string]string {
	ret := make(map[string]string)
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(filepath); err != nil {
		panic(err)
	}
	nodeinfo := doc.FindElement("./Nodeinfo")
	for i := 0; i < len(nodeinfo.ChildElements()); i++ {
		nodeindex := nodeinfo.SelectElement("NodeIndex" + strconv.Itoa(i))
		id := nodeindex.SelectAttr("NodeID").Value
		addr := nodeindex.SelectAttr("NodeAddr").Value
		ret[id] = addr
	}
	return ret
}
