package pbft

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"time"
)

var TpBegin time.Time

// PBFT 共识完成的通道标识,全局变量标识
var PFlag chan bool

// PBFT 共识消息通道
var Pmsg chan []Message

// 单机测试用全局主记账节点
var GlobalLeader string

//本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
var localMessagePool = []Message{}

type node struct {
	//节点ID
	nodeID string
	//节点监听地址
	addr string
	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte
}

type pbft struct {
	//节点信息
	node node
	//每笔请求自增序号
	sequenceID int
	//锁
	lock sync.Mutex
	//临时消息池，消息摘要对应消息本体
	messagePool map[string]BlockRequest
	//存放收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	prePareConfirmCount map[string]map[string]bool
	//存放收到的commit数量（至少需要收到并确认2f+1个），根据摘要来对应
	commitConfirmCount map[string]map[string]bool
	//该笔消息是否已进行Commit广播
	isCommitBordcast map[string]bool
	//该笔消息是否已对客户端进行Reply
	isReply map[string]bool
	//节点当前视图编号
	viewNum int
	// pbft主节点
	pbftLeaderNode string
}

func NewPBFT(nodeID, addr string) *pbft {
	p := new(pbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.node.rsaPrivKey = p.getPivKey(nodeID) //从生成的私钥文件处读取
	p.node.rsaPubKey = p.getPubKey(nodeID)  //从生成的私钥文件处读取
	p.sequenceID = 0
	p.messagePool = make(map[string]BlockRequest)
	p.prePareConfirmCount = make(map[string]map[string]bool)
	p.commitConfirmCount = make(map[string]map[string]bool)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	p.viewNum = 0         //初始视图编号为0
	p.pbftLeaderNode = "" //初始化主节点为空
	return p
}

func (p *pbft) handleRequest(data []byte) {
	//切割消息，根据消息命令调用不同的功能
	cmd, content := splitMessage(data)
	switch command(cmd) {
	case CRequest:
		p.handleClientRequest(content)
	case cPrePrepare:
		p.handlePrePrepare(content)
	case cPrepare:
		p.handlePrepare(content)
	case cCommit:
		p.handleCommit(content)
	}
}

//处理客户端发来的请求
func (p *pbft) handleClientRequest(content []byte) {
	//fmt.Println("节点已接收到客户端发来的request ...")
	if !CheckIsComitteeDone() {
		return
	}

	// 如果自己不是主节点则进行转发
	if p.pbftLeaderNode != p.node.nodeID {
		// 视图检查
		i := p.viewNum % PnodeCount
		p.setLeader(strconv.Itoa(i))
		//fmt.Println("主记账节点是", p.pbftLeaderNode, "正在重定向...")
		GlobalLeader = p.pbftLeaderNode
		data := JointMessage(CRequest, content)
		TcpDial(data, NodeTable[p.pbftLeaderNode])
		return
	}

	//Todo 主记账节点开启共识出块操作, 1,开启一个计时器，2，保存所有客户端请求

	//使用json解析出Request结构体
	r := new(Request)
	err := json.Unmarshal(content, r)
	if err != nil {
		log.Panic(err)
	}
	r.Message.Timestamp = r.Timestamp
	Br.Msg = append(Br.Msg, r.Message)
	Br.Time = time.Now().Unix()
	go func() {
		<-Ticker.C // 计时器
		fmt.Println("记账时间到：", Br.Time)
		//添加信息序号
		p.sequenceIDAdd()
		//获取消息摘要
		digest := getDigest(*Br)
		fmt.Println("已将request存入临时消息池")
		//存入临时消息池
		p.messagePool[digest] = *Br
		//主节点对消息摘要进行签名
		digestByte, _ := hex.DecodeString(digest)
		signInfo := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		//拼接成PrePrepare，准备发往follower节点
		pp := PrePrepare{*Br, p.viewNum, digest, p.sequenceID, p.node.nodeID, signInfo}
		b, err := json.Marshal(pp)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println("正在向其他节点进行进行PrePrepare广播 ...")
		//进行PrePrepare广播
		p.broadcast(cPrePrepare, b)
		fmt.Println("PrePrepare广播完成")
		// 清空 msg
		Br.Msg = Br.Msg[0:0]
	}()
}

//处理预准备消息
func (p *pbft) handlePrePrepare(content []byte) {
	fmt.Println("本节点已接收到主节点发来的PrePrepare ...")
	//	//使用json解析出PrePrepare结构体
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}

	if p.pbftLeaderNode == "" {
		p.pbftLeaderNode = pp.NodeID
		GlobalLeader = p.pbftLeaderNode
	}
	//获取主节点的公钥，用于数字签名验证
	//primaryNodePubKey := p.getPubKey("N0")
	primaryNodePubKey := p.getPubKey(pp.NodeID)
	digestByte, _ := hex.DecodeString(pp.Digest)
	/*if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		fmt.Println("信息摘要对不上，拒绝进行prepare广播", digest, pp.Digest)
		fmt.Println("消息内容：", pp.RequestMessage)
	} else */if p.sequenceID+1 != pp.SequenceID {
		fmt.Println("消息序号对不上，拒绝进行prepare广播", p.sequenceID, pp.SequenceID)
	} else if !p.RsaVerySignWithSha256(digestByte, pp.Sign, primaryNodePubKey) {
		fmt.Println("主节点签名验证失败！,拒绝进行prepare广播")
	} else {
		//序号赋值
		p.sequenceID = pp.SequenceID
		//将信息存入临时消息池
		fmt.Println("已将消息存入临时节点池")
		p.messagePool[pp.Digest] = pp.RequestMessage
		//节点使用私钥对其签名
		sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		//拼接成Prepare
		pre := Prepare{pp.ViewNum, pp.Digest, pp.SequenceID, p.node.nodeID, sign}
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		//进行准备阶段的广播
		fmt.Println("正在进行Prepare广播 ...")
		p.broadcast(cPrepare, bPre)
		fmt.Println("Prepare广播完成")
	}
}

//处理准备消息
func (p *pbft) handlePrepare(content []byte) {
	//使用json解析出Prepare结构体
	pre := new(Prepare)
	err := json.Unmarshal(content, pre)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("本节点已接收到%s节点发来的Prepare ... \n", pre.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.getPubKey(pre.NodeID)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if _, ok := p.messagePool[pre.Digest]; !ok {
		fmt.Println("当前临时消息池无此摘要，拒绝执行commit广播")
	} else if p.sequenceID != pre.SequenceID {
		fmt.Println("消息序号对不上，拒绝执行commit广播")
	} else if !p.RsaVerySignWithSha256(digestByte, pre.Sign, MessageNodePubKey) {
		fmt.Println("节点签名验证失败！,拒绝执行commit广播")
	} else {
		p.setPrePareConfirmMap(pre.Digest, pre.NodeID, true)
		count := 0
		for range p.prePareConfirmCount[pre.Digest] {
			count++
		}
		//TODO主节点可以收到 N-f-1 = 2f个prepare , 备份节点可以收到 N-1-1-f= 2f -1 个prepare(主节点不发，自己不给自己发)
		// ToDo 注： 一般说不小于 2f是包括自己本身的
		f := (PnodeCount - 1) / 3
		specifiedCount := 2*f - 1
		p.lock.Lock()
		if count >= specifiedCount && !p.isCommitBordcast[pre.Digest] {
			fmt.Println("本节点已收到至少2f个节点(包括本地节点)发来的Prepare信息 ...")
			//节点使用私钥对其签名
			sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
			c := Commit{pre.ViewNum, pre.Digest, pre.SequenceID, p.node.nodeID, sign}
			bc, err := json.Marshal(c)
			if err != nil {
				log.Panic(err)
			}
			//进行提交信息的广播
			fmt.Println("正在进行commit广播")
			p.broadcast(cCommit, bc)
			p.isCommitBordcast[pre.Digest] = true
			fmt.Println("commit广播完成")
		}
		p.lock.Unlock()
	}
}

//处理提交确认消息
func (p *pbft) handleCommit(content []byte) {
	//使用json解析出Commit结构体
	c := new(Commit)
	err := json.Unmarshal(content, c)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	MessageNodePubKey := p.getPubKey(c.NodeID)
	digestByte, _ := hex.DecodeString(c.Digest)
	if _, ok := p.prePareConfirmCount[c.Digest]; !ok {
		fmt.Println("当前prepare池无此摘要，拒绝将信息持久化到本地消息池")
	} else if p.sequenceID != c.SequenceID {
		fmt.Println("消息序号对不上，拒绝将信息持久化到本地消息池")
	} else if !p.RsaVerySignWithSha256(digestByte, c.Sign, MessageNodePubKey) {
		fmt.Println("节点签名验证失败！,拒绝将信息持久化到本地消息池")
	} else {
		p.setCommitConfirmMap(c.Digest, c.NodeID, true)
		count := 0
		for range p.commitConfirmCount[c.Digest] {
			count++
		}
		//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！
		f := (PnodeCount - 1) / 3 // 每个节点至少收到 N-f -1 个commit（不包括自己）
		p.lock.Lock()
		if count >= 2*f && !p.isReply[c.Digest] && p.isCommitBordcast[c.Digest] {
			fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
			//将消息信息，提交到本地消息池中！
			localMessagePool = p.messagePool[c.Digest].Msg // 没有必要了
			//info := p.node.nodeID + "节点已将msgid:" + strconv.Itoa(p.messagePool[c.Digest].ID) + "存入本地消息池中,消息内容为：" + p.messagePool[c.Digest].Content
			// 通道 告诉 raft 共识完成了
			PFlag <- true
			Pmsg <- p.messagePool[c.Digest].Msg
			p.isReply[c.Digest] = true
		}
		p.lock.Unlock()
	}
}

//序号累加
func (p *pbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

//向除自己外的其他节点进行广播
func (p *pbft) broadcast(cmd command, content []byte) {

	if !CheckIsComitteeDone() {
		fmt.Println("委员会节点读取失败")
		return
	}

	for i, addr := range NodeTable {
		if i == p.node.nodeID {
			continue
		}
		message := JointMessage(cmd, content)
		//go TcpDial(message, NodeTable[i])
		go TcpDial(message, addr)
	}
}

// 委员会 PBFT 共识节点不可小于4
func CheckIsComitteeDone() bool {

	// 减少 读json的频率
	PnodeCount = len(NodeTable)
	if PnodeCount >= 4 {
		return true
	}

	// 读配置文件
	var lock sync.Mutex
	lock.Lock()
	readjson("council.json", NodeTable)
	lock.Unlock()

	PnodeCount = len(NodeTable)
	if PnodeCount < 4 {
		return false
	}
	return true
}

func readjson(jsonFile string, nodetable map[string]string) error {
	byteValue, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return err
	}

	// We have known the outer json object is a map, so we define  result as map.
	// otherwise, result could be defined as slice if outer is an array
	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		return err
	}

	// handle peers
	nodes := result["nodes"].([]interface{})
	for _, node := range nodes {
		m := node.(map[string]interface{})
		if _, exists := m["Group"]; exists {
			nodeid := m["nodeid"].(string)
			nodeaddr := m["nodeaddr"].(string)
			nodetable[nodeid] = nodeaddr
		}
	}
	return nil
}

//为多重映射开辟赋值
func (p *pbft) setPrePareConfirmMap(val, val2 string, b bool) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = make(map[string]bool)
	}
	p.prePareConfirmCount[val][val2] = b
}

//为多重映射开辟赋值
func (p *pbft) setCommitConfirmMap(val, val2 string, b bool) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = make(map[string]bool)
	}
	p.commitConfirmCount[val][val2] = b
}

//传入节点编号， 获取对应的公钥
func (p *pbft) getPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}

//传入节点编号， 获取对应的私钥
func (p *pbft) getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

func (p *pbft) setLeader(i string) error {
	// 读json 文件
	byteValue, err := ioutil.ReadFile("council.json")
	if err != nil {
		return err
	}

	// We have known the outer json object is a map, so we define  result as map.
	// otherwise, result could be defined as slice if outer is an array
	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		return err
	}

	// handle peers
	nodes := result["nodes"].([]interface{})
	for _, node := range nodes {
		m := node.(map[string]interface{})
		if index, exists := m["Group"]; exists {
			if index.(string) == i {
				nodeid := m["nodeid"].(string)
				p.pbftLeaderNode = nodeid
				break
			}
		}
	}
	return nil
}
