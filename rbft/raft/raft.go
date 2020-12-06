package raft

import (
	bk "consensus/rbft/block"
	"consensus/rbft/pbft"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

)

//声明raft节点类型
type Raft struct {
	node *NodeInfo
	//本节点获得的投票数
	vote int
	//线程锁
	lock sync.Mutex
	//节点编号
	me string
	//当前任期
	currentTerm int
	//为哪个节点投票
	votedFor string
	//当前节点状态
	//0 follower  1 candidate  2 leader
	state int
	//发送最后一条消息的时间
	lastMessageTime int64
	//发送最后一条消息的时间
	lastHeartBeatTime int64
	//当前节点的领导
	currentLeader string
	//心跳超时时间(单位：秒)
	timeout int
	//接收投票成功通道
	voteCh chan bool
	// log 确认通道
	ConfirmCh chan bool
	//心跳信号
	heartBeat chan bool
}

const (
	FOLLOWER  = 0
	CANDIDATE = 1
	LEADER    = 2
)

type NodeInfo struct {
	ID   string
	Addr string
}

func NewRaft(id, addr string) *Raft {
	node := new(NodeInfo)
	node.ID = id
	node.Addr = addr

	rf := new(Raft)
	//节点信息
	rf.node = node
	//当前节点获得票数
	rf.setVote(0)
	//编号
	rf.me = id
	//给节点投票，初始化 -1 给谁都不投
	rf.setVoteFor("-1")
	//0 follower
	rf.setStatus(FOLLOWER)
	//最后一次心跳检测时间
	rf.lastHeartBeatTime = 0
	rf.timeout = heartBeatTimeout
	//最初没有领导
	rf.setCurrentLeader("-1")
	//设置任期
	rf.setTerm(0)
	//投票通道
	rf.voteCh = make(chan bool)
	// 消息确认通道
	rf.ConfirmCh = make(chan bool)
	//心跳通道
	rf.heartBeat = make(chan bool)
	return rf
}

//修改节点为候选人状态
func (rf *Raft) becomeCandidate() bool {
	r := randRange(1500, 5000)
	//休眠随机时间后，再开始成为候选人
	time.Sleep(time.Duration(r) * time.Millisecond)
	//如果发现本节点已经投过票，或者已经存在领导者，则不用变身候选人状态
	if rf.state == 0 && rf.currentLeader == "-1" && rf.votedFor == "-1" {
		//将节点状态变为1
		rf.setStatus(CANDIDATE)
		//设置为哪个节点投票
		rf.setVoteFor(rf.me)
		//节点任期加1
		rf.setTerm(rf.currentTerm + 1)
		//当前没有领导
		rf.setCurrentLeader("-1")
		fmt.Println("本节点已变更为候选人状态,Term", rf.currentTerm)
		//为自己投票
		rf.voteAdd()
		fmt.Printf("当前得票数：%d\n", rf.vote)
		//开启选举通道
		return true
	}
	return false
}

//进行选举
func (rf *Raft) election() bool {
	fmt.Println("Leader 选举开始...")
	// 广播选票
	r := new(Nonmsg)
	r.Voteflag = false
	r.Node.ID = rf.node.ID
	r.Node.Addr = rf.node.Addr
	r.Term = rf.currentTerm
	r.Timestamp = time.Now().UnixNano()

	mssg, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Println("mssg:", string(mssg))
	fmt.Println("正在广播选票...")
	rf.broadcast(vote, mssg)

	for {
		select {
		case <-time.After(time.Second * time.Duration(timeout)):
			fmt.Println("领导者选举超时，节点变更为追随者状态\n")
			rf.reDefault()
			return false
		case ok := <-rf.voteCh:
			if ok {
				rf.voteAdd()
				fmt.Printf("获得来自其他节点的投票，当前得票数：%d\n", rf.vote)
			}
			if rf.vote > raftCount/2 && rf.currentLeader == "-1" {
				fmt.Println("获得超过网络节点二分之一的得票数，本节点被选举成为了leader")
				//节点状态变为2，代表leader
				rf.setStatus(LEADER)
				//当前领导者为自己
				rf.setCurrentLeader(rf.me)
				// 开始组建委员会，向其他节点广播自己leader的身份，开启PBFT 监听

				StartComittee(rf.me)
				fmt.Println(rf.me, "向其他节点进行广播leader 确认...")
				rf.broadcast(leader, mssg)
				// 开启心跳检测
				rf.heartBeat <- true
				// 开启日志同步监听
				go rf.handleLog()
				return true
			}
		}
	}
}

func (rf *Raft) handleRequest(data []byte) {
	//切割消息，根据消息命令调用不同的功能
	cmd, content := splitMessage(data)
	switch command(cmd) {
	case vote:
		rf.handlevote(content)
	case leader:
		rf.handleleaderconfirm(content)
	case heartbeat:
		rf.handleheartbeat(content)
	case Cmessage:
		rf.handleCmsg(content)
	case Lmessage:
		rf.handleLmsg(content)
	case commit:
		rf.handleCommit(content)
	}
}

// 处理请求投票的消息, 看来必须要有 rpc 远程调用了
func (rf *Raft) handlevote(content []byte) {
	// 解析 content
	recv := new(Nonmsg)
	err := json.Unmarshal(content, recv)
	if err != nil {
		log.Panic(err)
	}
	if !recv.Voteflag {
		if rf.currentTerm != recv.Term {
			// 不在一个任期，那原来的投票无效
			rf.setVoteFor("-1")
			// 设置到跟 candidate 同一个任期
			rf.setTerm(recv.Term)
		}
		// 有领导者了不投票，不是follower 状态也不投票，同一个任期，只投一次票
		if rf.currentLeader != "-1" || rf.state != FOLLOWER || rf.votedFor != "-1" {
			fmt.Printf("leadr %s--state--%dvotedfor %s", rf.currentLeader, rf.state, rf.votedFor)
			return
		} else {
			rf.setVoteFor(recv.Node.ID)
			recv.Voteflag = true
			mssg, err := json.Marshal(recv)
			if err != nil {
				log.Panic(err)
			}
			msg := jointMessage(vote, mssg)
			go tcpDial(msg, recv.Node.Addr)
			fmt.Printf("投票成功，已投%s节点\n", recv.Node.ID)
		}
	} else {
		rf.voteCh <- true
	}

}

// 收到领导者节点的响应
func (rf *Raft) handleLmsg(content []byte) {
	recv := new(Logmsg)
	err := json.Unmarshal(content, recv)
	if err != nil {
		log.Panic(err)
	}
	if !recv.ConfirmFlag {
		// follower 共识从这里开始
		// tBegin = time.Now()
		recv.ConfirmFlag = true
		// follower 临时保存log
		mssg, err := json.Marshal(recv)
		if err != nil {
			log.Panic(err)
		}
		msg := jointMessage(Lmessage, mssg)
		go tcpDial(msg, recv.Node.Addr) // 回复leader 收到了
		fmt.Printf("本节点已收到leader日志已回复，待确认\n")
	} else {
		rf.ConfirmCh <- true
	}
}

func (rf *Raft) handleleaderconfirm(content []byte) {
	recv := new(Nonmsg)
	err := json.Unmarshal(content, recv)
	if err != nil {
		log.Panic(err)
	}
	rf.setCurrentLeader(recv.Node.ID)
	fmt.Println("已发现网络中的领导节点，", recv.Node.ID, "成为了领导者！")
	rf.reDefault()
}

//心跳检测方法
func (rf *Raft) heartbeat() {
	//如果收到通道开启的信息,将会向其他节点进行固定频率的心跳检测
	r := new(Nonmsg)
	r.Voteflag = false
	r.Node.ID = rf.node.ID
	r.Node.Addr = rf.node.Addr
	r.Timestamp = time.Now().UnixNano()

	mssg, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	if <-rf.heartBeat {
		for {
			//fmt.Println("本节点开始发送心跳检测...")
			rf.broadcast(heartbeat, mssg)
			rf.lastHeartBeatTime = millisecond()
			time.Sleep(time.Second * time.Duration(heartBeatTimes)) // 每 3秒发送一次心跳检测
		}
	}
}

func (rf *Raft) handleheartbeat(content []byte) {
	recv := new(Nonmsg)
	err := json.Unmarshal(content, recv)
	if err != nil {
		log.Panic(err)
	}
	rf.setCurrentLeader(recv.Node.ID)
	rf.lastHeartBeatTime = millisecond()
	fmt.Printf("接收到来自领导节点%s的心跳检测\n", recv.Node.ID)
	fmt.Printf("当前时间为:%d\n\n", millisecond())
}

func (rf *Raft) handleCmsg(content []byte) {
	fmt.Println("收到客户端请求")

	// 广播 lmessage 加上leader 的id 信息
	r := new(Logmsg)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}

	r.Node.ID = rf.node.ID
	r.Node.Addr = rf.node.Addr

	logmsg, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	rf.broadcast(Lmessage, logmsg)
	num := 0
	for {
		// 判断 follower 的回复数，满足条件 Raft验证通过commit
		select {
		case <-time.After(time.Second * time.Duration(10)):
			fmt.Println("日志同步超时")
			return
		case ok := <-rf.ConfirmCh:
			if ok {
				num++
			}
			// 收到 半数以上节点，默认自己已收到，所以减1
			if num >= (raftCount-1)/2 {
				fmt.Printf("全网已超过半数节点接收到消息：\nraft验证通过，准备同步至follower...")
				// 创块 正常讲确实每个节点都应该创块，更新本地账本，但是需要注意单机测试只有一个账本，这里要防止重复记账的问题
				if pbft.GlobalLeader == rf.node.ID {
					rf.AddBlock(r.Content)
				}
				rf.lastMessageTime = millisecond()
				rf.broadcast(commit, content)
				return
			}
		}
	}

	// ToDo

}

// follower 收到 commit 直接写入statemachine 不需要回复
func (rf *Raft) handleCommit(content []byte) {
	fmt.Println("收到commit,正在写入状态机，共识完毕...")
	rf.lastMessageTime = millisecond()
	// tEnd = time.Now()
	// fmt.Println("耗时：", tEnd.Sub(tBegin))
}

//产生随机值
func randRange(min, max int64) int64 {
	//用于心跳信号的时间
	rand.Seed(time.Now().UnixNano())
	return rand.Int63n(max-min) + min
}

//获取当前时间的毫秒数
func millisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//设置任期
func (rf *Raft) setTerm(term int) {
	rf.lock.Lock()
	rf.currentTerm = term
	rf.lock.Unlock()
}

//设置为谁投票
func (rf *Raft) setVoteFor(id string) {
	rf.lock.Lock()
	rf.votedFor = id
	rf.lock.Unlock()
}

//设置当前领导者
func (rf *Raft) setCurrentLeader(leader string) {
	rf.lock.Lock()
	rf.currentLeader = leader
	rf.lock.Unlock()
}

//设置当前领导者
func (rf *Raft) setStatus(state int) {
	rf.lock.Lock()
	rf.state = state
	rf.lock.Unlock()
}

//投票累加
func (rf *Raft) voteAdd() {
	rf.lock.Lock()
	rf.vote++
	rf.lock.Unlock()
}

//设置投票数量
func (rf *Raft) setVote(num int) {
	rf.lock.Lock()
	rf.vote = num
	rf.lock.Unlock()
}

//恢复默认设置
func (rf *Raft) reDefault() {
	rf.setVote(0)
	//rf.currentLeader = "-1"
	rf.setVoteFor("-1")
	rf.setStatus(FOLLOWER)
}

// 创建区块
func (rf *Raft) AddBlock(data []bk.Bdata) {
	bc := bk.NewBlockchain()
	defer bc.Db.Close()
	bc.AddBlock2(data)
}
