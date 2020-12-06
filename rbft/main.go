package main

import (
	bk "consensus/rbft/block"
	"consensus/rbft/raft"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var timeStart, timeEnd time.Time

func main() {
	cmd := os.Args[1]
	switch cmd {
	case "runnode":
		nodeid := os.Args[2]
		fmt.Println("启动节点", nodeid)

		raft.Raftgo(nodeid)
	case "client":
		fmt.Printf("启动客户端\n")
		clientRun()
	case "printchain":
		printchain()
	}
}

// 产生一个随机字符串
func GetRandom() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err == nil {
		//fmt.Println("rand.Read：", base64.URLEncoding.EncodeToString(b))
		return base64.URLEncoding.EncodeToString(b)
	}
	return ""
}

// 并发向多个客户端发请求测算tps http://127.0.0.1:8080/req
func clientRun() {
	go send("http://127.0.0.1:8080/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8081/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8082/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8083/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8084/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8085/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8086/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8087/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8088/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8089/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8090/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8091/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8092/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8093/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8094/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8095/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8096/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8097/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8098/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8099/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8100/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8101/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8102/req")
	time.Sleep(50 * time.Millisecond)
	go send("http://127.0.0.1:8103/req")
	select {}
}

func send(addr string) {
	i := 0
	t := time.Now()
	for {
		time.Sleep(50 * time.Millisecond)
		msg := "test" + strconv.Itoa(i)
		timeStart = time.Now()
		res, err := http.PostForm(addr, url.Values{"message": {msg}})
		if err != nil {
			fmt.Println("连接失败：", err.Error())
			return
		}
		res.Body.Close()
		i++
		if i > 1000 { // 太大了不好，每隔1000重置一次，最大值 65535 如果用int64 可达18446744073709551615
			i = 0
		}
		if time.Since(t) >= 5*time.Minute { // 2分钟后断开
			fmt.Println("时间到，TPS 测算完毕")
			break
		}
	}
}

//本都测试用
func printchain() {
	bc := bk.NewBlockchain()
	defer bc.Db.Close()

	bci := bc.Iterator()
	for {
		block := bci.Next()

		fmt.Printf("Timestamp:%v\n", block.Timestamp)
		fmt.Printf("Prev. hash:%x\n", block.PrevBlockHash)
		// decode blockdata
		Data := bk.DecodeBlockData(block.Data)
		dt := Data.([]bk.Bdata)
		dtlen := len(dt)
		fmt.Println("DataCount:", dtlen)
		if dtlen >= 3 {
			fmt.Println("Last 3 Data:", dt[dtlen-3], dt[dtlen-2], dt[dtlen-1])
			fmt.Println("处理时间： s", (float64)(block.Timestamp-dt[dtlen-1].Timestamp)*0.000000001)
		} else {
			fmt.Println("Data:", dt)
		}

		fmt.Printf("Hash:%x\n", block.Hash)
		fmt.Printf("\n\n")
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

}
