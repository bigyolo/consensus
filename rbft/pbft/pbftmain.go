package pbft

import "time"
var PnodeCount int

//节点池，主要用来存储监听地址
var NodeTable map[string]string

// PBFT 的节点池不应该是直接初始化的，而应该通过Raft 选举确定
//PBFT 主节点记账计时器
var Ticker *time.Ticker
var Br *BlockRequest
func init() {
	NodeTable = make(map[string]string)
	// PBFT 共识完成的通道标识，设置带缓冲，异步处理
	PFlag = make(chan bool)
	Pmsg = make(chan []Message)
	Ticker = time.NewTicker(10 * time.Second)
	Br = new(BlockRequest)
}
