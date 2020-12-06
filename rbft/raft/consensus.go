package raft

import (
	"consensus/rbft/pbft"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

)

//等待节点访问
func (rf *Raft) getRequest(writer http.ResponseWriter, request *http.Request) {
	request.ParseForm()
	//http://localhost:8080/req?message=ohmygod
	if len(request.Form["message"]) > 0 && rf.currentLeader != "-1" {
		//fmt.Println("当前的领导者是", rf.currentLeader)
		message := request.Form["message"][0]
		// 消息重定向到leader，开启pbft 共识
		startPbftConsensus(message, rf.currentLeader, request.RemoteAddr)
		writer.Write([]byte("ok!!!"))
	}
}

// raft进行日志同步
func (rf *Raft) handleLog() {
	fmt.Println(rf.me, "handleLog准备完毕...")
	for {
		if rf.currentLeader != "-1" && <-pbft.PFlag {
			msg := <-pbft.Pmsg
			//fmt.Println("组内共识:", msg)
			r := new(Logmsg)
			r.Content = msg
			r.Timestamp = time.Now().UnixNano()
			//消息内容就是用户的输入
			br, err := json.Marshal(r)
			if err != nil {
				log.Panic(err)
			}

			content := jointMessage(Cmessage, br)
			tcpDial(content, nodeTable[rf.currentLeader])
		}
	}
}

func startPbftConsensus(data, node, cid string) {
	if false == pbft.CheckIsComitteeDone() {
		fmt.Println("委员会尚未组建完成，请检查")
		return
	}
	//fmt.Println("委员会共识开始")
	r := new(pbft.Request)
	r.Timestamp = time.Now().UnixNano()
	r.ClientId = ""
	r.Message.Timestamp = r.Timestamp
	//消息内容就是用户的输入
	r.Message.Msg = data
	br, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	content := pbft.JointMessage(pbft.CRequest, br)
	pbft.TcpDial(content, pbft.NodeTable[node])

}

func (rf *Raft) httpListen(id string) {
	var addr string
	switch id {
	case "N0":
		addr = ":8080"
	case "N1":
		addr = ":8081"
	case "N2":
		addr = ":8082"
	case "N3":
		addr = ":8083"
	case "N4":
		addr = ":8084"
	case "N5":
		addr = ":8085"
	case "N6":
		addr = ":8086"
	case "N7":
		addr = ":8087"
	case "N8":
		addr = ":8088"
	case "N9":
		addr = ":8089"
	case "N10":
		addr = ":8090"
	case "N11":
		addr = ":8091"
	case "N12":
		addr = ":8092"
	case "N13":
		addr = ":8093"
	case "N14":
		addr = ":8094"
	case "N15":
		addr = ":8095"
	case "N16":
		addr = ":8096"
	case "N17":
		addr = ":8097"
	case "N18":
		addr = ":8098"
	case "N19":
		addr = ":8099"
	case "N20":
		addr = ":8100"
	case "N21":
		addr = ":8101"
	case "N22":
		addr = ":8102"
	case "N23":
		addr = ":8103"
	}

	//创建getRequest()回调方法
	http.HandleFunc("/req", rf.getRequest)
	fmt.Println("监听", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println(err)
		return
	}
}
