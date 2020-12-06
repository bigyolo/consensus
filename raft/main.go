package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/beevik/etree"

)

//定义节点数量
var raftCount int

//节点池
var nodeTable map[string]string

//选举超时时间（单位：秒）
var timeout = 3

//心跳检测超时时间
var heartBeatTimeout = 7

//心跳检测频率（单位：秒）
var heartBeatTimes = 3

// 耗时统计
var tbegin time.Time
var tend time.Time

func main() {
	//stopper := profile.Start(profile.CPUProfile, profile.ProfilePath("."))  性能测试工具
	nodeTable = make(map[string]string)
	nodeTable = readnodeTotable("./node.xml")

	raftCount = len(nodeTable)
	id := os.Args[1]
	if id == "client" {
		clientSendMessage()
		return
	}
	raft := NewRaft(id, nodeTable[id])
	// 测试一下客户端并发
	// go raft.httpListen(id)
	//启用RPC,注册raft
	go raft.tcpListen()
	//开启心跳检测
	go raft.heartbeat()

Circle:
	/*开启选举   本段代码嵌套循环，cpu占用很高，不采用
	go func() {
		for {
			//成为候选人节点
			if raft.becomeCandidate() {
				//成为后选人节点后 向其他节点要选票来进行选举
				if raft.election() {
					stopper.Stop()
					break
				} else {
					continue
				}
			} else {
				break
			}
		}
	}()
	*/
	// 解决cpu占用过高的问题，但是需要大量测试
	if raft.becomeCandidate() {
		if !raft.election() {
			goto Circle
		}
	} else {
		goto Circle
	}

	for {
		if raft.lastHeartBeatTime != 0 && (millisecond()-raft.lastHeartBeatTime) > int64(raft.timeout*1000) {
			fmt.Printf("心跳检测超时，已超过%d秒\n", raft.timeout)
			fmt.Println("即将重新开启选举")
			raft.reDefault()
			raft.setCurrentLeader("-1")
			raft.lastHeartBeatTime = 0
			goto Circle
		}
	}
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
