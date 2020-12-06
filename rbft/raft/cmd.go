package raft

import (
	"consensus/rbft/pbft"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"github.com/beevik/etree"

)

type command string

const prefixCMDLength = 12

// 请求投票、leader确认、heartbeat\commit的消息体
type Nonmsg struct {
	Voteflag  bool
	Node      NodeInfo
	Term      int
	Timestamp int64
}

type Logmsg struct {
	ConfirmFlag bool
	Node        NodeInfo
	Content     []pbft.Message
	Timestamp   int64
}

const (
	vote      command = "vote"      //广播选票
	heartbeat command = "heartbeat" //广播心跳
	leader    command = "leader"    //广播leader身份message
	Cmessage  command = "Cmessage"  //客户端请求message
	Lmessage  command = "Lmessage"
	commit    command = "commit" //广播commit
)

type Message struct {
	Msg   []pbft.Message
	MsgID int64
}

//默认前十二位为命令名称
func jointMessage(cmd command, content []byte) []byte {
	b := make([]byte, prefixCMDLength)
	for i, v := range []byte(cmd) {
		b[i] = v
	}
	joint := make([]byte, 0)
	joint = append(b, content...)
	return joint
}

//默认前十二位为命令名称
func splitMessage(message []byte) (cmd string, content []byte) {
	cmdBytes := message[:prefixCMDLength]
	newCMDBytes := make([]byte, 0)
	for _, v := range cmdBytes {
		if v != byte(0) {
			newCMDBytes = append(newCMDBytes, v)
		}
	}
	cmd = string(newCMDBytes)
	content = message[prefixCMDLength:]
	return
}

// StartComittee 组建委员会，leader 节点广播自己地址，组建pbft 委员会
func StartComittee(nodeid string) {
	fmt.Println("leader", nodeid, "加入PBFT委员会")
	pbft.GenRsaKeys(nodeid)

	var nodeAddr string
	switch groupNum {
	case 0:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5000"
	case 1:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5001"
	case 2:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5002"
	case 3:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5003"
	case 4:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5004"
	case 5:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5005"
	case 6:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5006"
	case 7:
		nodeAddr = nodeTable[nodeid][0:len(nodeTable[nodeid])-4] + "5007"
	}

	p := pbft.NewPBFT(nodeid, nodeAddr)
	go p.TcpListen()

	// 更新委员会配置文件,加读写锁
	updateCouncil("council.json", nodeid, nodeAddr, groupNum)
}

func updateCouncil(jsonFile, nodeid, nodeAddr string, num int) error {
	lock.Lock()
	byteValue, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Println(err)
		return err
	}
	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		log.Println(err)
		return err
	}
	nodes := result["nodes"].([]interface{})

	for _, node := range nodes {
		m := node.(map[string]interface{})
		if num, exists := m["Group"]; exists {
			if num == strconv.Itoa(groupNum) {
				m["nodeid"] = nodeid
				m["nodeaddr"] = nodeAddr
			}
		} else {
			fmt.Println("no exists")
		}
	}

	// Convert golang object back to byte
	byteValue, err = json.Marshal(result)
	if err != nil {
		log.Println(err)
		return err
	}

	// Write back to file
	err = ioutil.WriteFile(jsonFile, byteValue, 0644)
	lock.Unlock()
	return err
}

//ReadXMLToNodeTable 从配置文件获取分组
func ReadXMLToNodeTable(filepath, id string) (map[string]string, int) {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(filepath); err != nil {
		panic(err)
	}
	root := doc.FindElement("./Nodeinfo")

	for i := range root.ChildElements() {
		Fchild := root.SelectElement("Group" + strconv.Itoa(i))
		for j := range Fchild.ChildElements() {
			Schild := Fchild.SelectElement("NodeIndex" + strconv.Itoa(j))
			if nodeid := Schild.SelectAttr("NodeID"); nodeid != nil {
				if nodeid.Value == id {
					return WriteNodeTable(Fchild), i
				}
			}
		}
	}
	return nil, -1
}

// WriteNodeTable n初始化 nodetable
func WriteNodeTable(Group *etree.Element) map[string]string {
	for i := range Group.ChildElements() {
		nodedindex := Group.SelectElement("NodeIndex" + strconv.Itoa(i))
		nodeid := nodedindex.SelectAttr("NodeID")
		nodeaddr := nodedindex.SelectAttr("NodeAddr")
		nodeTable[nodeid.Value] = nodeaddr.Value
	}
	return nodeTable
}
