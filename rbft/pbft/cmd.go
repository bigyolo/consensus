package pbft

import (
	bk "consensus/rbft/block"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"log"

)

type Message = (bk.Bdata)

//<REQUEST,o,t,c>
type Request struct {
	Message
	Timestamp int64
	ClientId  string
}

//区块请求 多条信息
type BlockRequest struct {
	Msg  []Message
	Time int64
}

//<<PRE-PREPARE,v,n,d>,m>
type PrePrepare struct {
	RequestMessage BlockRequest
	ViewNum        int
	Digest         string
	SequenceID     int
	NodeID         string
	Sign           []byte
}

//<PREPARE,v,n,d,i>
type Prepare struct {
	ViewNum    int
	Digest     string
	SequenceID int
	NodeID     string
	Sign       []byte
}

//<COMMIT,v,n,D(m),i>
type Commit struct {
	ViewNum    int
	Digest     string
	SequenceID int
	NodeID     string
	Sign       []byte
}

//<REPLY,v,t,c,i,r>
type Reply struct {
	ViewNum   int
	MessageID int
	NodeID    string
	Result    bool
}

type Council struct {
	Id    string
	Addr  string
	Group int
}

const prefixCMDLength = 12

type command string

const (
	CRequest    command = "request"
	cPrePrepare command = "preprepare"
	cPrepare    command = "prepare"
	cCommit     command = "commit"
)

//默认前十二位为命令名称
func JointMessage(cmd command, content []byte) []byte {
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

//对消息详情进行摘要
// func getDigest(request Request) string {
// 	b, err := json.Marshal(request)
// 	if err != nil {
// 		log.Panic(err)
// 	}
// 	hash := sha256.Sum256(b)
// 	//进行十六进制字符串编码
// 	return hex.EncodeToString(hash[:])
// }

func getDigest(request BlockRequest) string {
	b, err := json.Marshal(request)
	if err != nil {
		log.Panic(err)
	}
	//hash := sha256.Sum256(b) //输入小于2^64
	hash := sha512.Sum512(b) // 2^128

	//进行十六进制字符串编码
	return hex.EncodeToString(hash[:])
}
