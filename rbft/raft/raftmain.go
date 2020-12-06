package raft

import (
	"fmt"
	"sync"
	"time"
)

//定义节点数量
var raftCount int

//Raft 节点池组，大写可以在外部通过包引用，如果不需要应该采用小写格式
var nodeTable map[string]string

//选举超时时间（单位：秒）
var timeout = 3

//心跳检测超时时间
var heartBeatTimeout = 7

//心跳检测频率（单位：秒）
var heartBeatTimes = 3

//用于存储消息
var MessageStore = make(map[int]string)

// 节点所在分组编号
var groupNum = -1

// 文件读写锁
var lock sync.Mutex

//节点配置文件路径
const xmlpath = "./nodegroup.xml"

func Raftgo(node string) {
	id := node
	nodeTable = make(map[string]string)
	nodeTable, groupNum = ReadXMLToNodeTable(xmlpath, id)
	if nodeTable == nil || groupNum == -1 {
		fmt.Println("未定义此节点:", id, groupNum)
		return
	}
	// 分组节点数量
	raftCount = len(nodeTable)
	raft := NewRaft(id, nodeTable[id])

	//启用RPC,注册raft
	go raft.tcpListen()
	//开启心跳检测
	go raft.heartbeat()
	//开启一个Http监听，这里应该所有节点都有监听，因为所有节点都可能收到客户端的请求并转发给组内leader
	go raft.httpListen(id)

Circle:
	//开启选举
	if raft.becomeCandidate() {
		if !raft.election() {
			goto Circle
		}
	} else {
		goto Circle
	}

	//进行心跳检测
	for {
		//0.5秒检测一次
		time.Sleep(time.Millisecond * 5000)
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
