package main

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
	Content     string
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
