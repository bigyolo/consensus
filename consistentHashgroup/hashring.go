package main

import (
	"consistentHash/chash1"
	"consistentHash/etree"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// 一致性hash算法进行节点分组
/************设计思路**********
1，随机函数，根据随机函数产生的随机值生成hash环上的节点
2，根据节点的ip地址+随机数 进行hash运算确定节点所处的位置，从而确定分组
3，对监督节点进行ip地址加随机数进行k次运算，从而使监督节点同时分配在k个组内
4，分组完毕
***********************/
const (
	nodeNum    = 20 //普通节点数
	groupNum   = 10 // 分组数
	everygroup = 5  // 设置每几组分一个监督节点
	xmlpath    = "./gnode.xml"
)

// Makehashring 确定分组
func Makehashring() *chash1.Consistent {
	cHashRing := chash1.NewConsistent()

	// 确定分组，初始化hash环
	for i := 0; i < groupNum; i++ {
		node := chash1.NewNode(i, chash1.GetRandom(), 1)
		cHashRing.Add(node)
	}
	return cHashRing
}

// 初始化节点
func initnode(nodetable, svnodes map[string]string, svnodeNum int) {
	for i := 0; i < nodeNum; i++ {
		nodeid := fmt.Sprintf("N%d", i)
		nodeaddr := fmt.Sprintf("192.168.242.%03d:%04d", i, i)
		nodetable[nodeid] = nodeaddr
	}
	for i := 0; i < svnodeNum; i++ {
		nodeid := fmt.Sprintf("SN%d", i)
		nodeaddr := fmt.Sprintf("127.0.0.%d:%04d", i, i)
		svnodes[nodeid] = nodeaddr
	}
}

//初始化一个空的xml文件
func initxml(doc *etree.Document, filepath string) {
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	Nodeinfo := doc.CreateElement("Nodeinfo")
	Nodeinfo.CreateAttr("Chinesesimple", "节点分组信息管理")
	for i := 0; i < groupNum; i++ {
		si := strconv.Itoa(i)
		Group := Nodeinfo.CreateElement("Group" + si)
		Group.CreateAttr("Chinesesimple", "第"+si+"组")
	}
	doc.Indent(4)
	if err := doc.WriteToFile(filepath); err != nil {
		panic(err)
	}
	doc.WriteToFile(filepath)
}

// SaveNode 将节点保存至对应的分组
// 一致性Hash 保证的节点分组的分配保持均匀，但要增加一个处理，保障每个组内至少有三个节点，防止raft不能正常工作
/********监督节点的分配存在以下问题
1, 一个监督节点在同一个组内只可存在一个副本，要去重处理
2，保证每一个组里必须有一个监督节点

//Todo 1, 通过增加虚拟节点的数量配置可以大幅度解决分配不均的问题 2，通过人工干预可以尽量使每个监督节点至少分在一个组内
// Todo 优化，将监督节点优先分配到没有监督节点的分组中去
***********/
func SaveNode(term int, doc *etree.Document, id, addr string, num int, svflag bool) bool {
	// 找到groupnum 分组
	snum := strconv.Itoa(num)
	Group := doc.FindElement("./Nodeinfo/Group" + snum)
	if Group == nil {
		fmt.Println("没有这个分组")
	}

	svcount := 0 //同一个组内的监督节点计数
	if svflag && len(Group.ChildElements()) != 0 {
		for i := range Group.ChildElements() {
			node := Group.SelectElement("Nodeindex" + strconv.Itoa(i+1))
			nodeid := node.SelectAttr("NodeID")
			if nodeid.Value == id {
				// 已经有了，则不操作.普通节点也是
				return false
			}
			sv := node.SelectAttr("SvFlag")
			if term == 0 && (sv != nil && sv.Value == "true") {
				return false
			}

			//ToDo 如何保证每组至少有一个监督节点
			// 增加一个控制，一个组内最多两个监督节点，人工干预监督节点的分配
			if sv := node.SelectAttr("SvFlag"); sv != nil {
				svcount++
			}
			if svcount >= 2 {
				//fmt.Println("no more new svnode")
				return false
			}
		}
	}

	snum = strconv.Itoa(len(Group.ChildElements()) + 1)
	nodeindex := Group.CreateElement("Nodeindex" + snum)
	nodeindex.CreateAttr("NodeID", id)
	nodeindex.CreateAttr("NodeAddr", addr)
	nodeindex.CreateAttr("EnableUsed", "true")
	if svflag {
		nodeindex.CreateAttr("SvFlag", "true")
	}
	doc.Indent(4)
	return true
}

// CheckIsDivided 对分组进行合理性检查 1，每个组内的节点数不得小于4 2，每个组内必须有一个监督节点
func CheckIsDivided(doc *etree.Document) bool {
	ret := false
	for i := 0; i < groupNum; i++ {
		Group := doc.FindElement("./Nodeinfo/Group" + strconv.Itoa(i))
		// if len(Group.ChildElements()) < 4 {
		// 	return false
		// }
		ret = false
		for j := 0; j < len(Group.ChildElements()); j++ {
			index := Group.SelectElement("Nodeindex" + strconv.Itoa(j+1))
			if svflag := index.SelectAttr("SvFlag"); svflag != nil {
				if svflag.Value == "true" {
					ret = true
					break //可以跳出nodeindex 循环，检查下一个group
				}
			}
		}
		if !ret {
			return ret
		}
	}
	return ret
}

func NodeDistribution() *etree.Document {
	cHashRing := Makehashring()

	nodetable := make(map[string]string)
	svnodes := make(map[string]string)
	svnodeNum := 0
	if groupNum%everygroup == 0 {
		svnodeNum = groupNum / everygroup
	} else {
		svnodeNum = groupNum/everygroup + 1
	}

	initnode(nodetable, svnodes, svnodeNum)

Check:
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(xmlpath); err != nil {
		initxml(doc, xmlpath)
	}

	for i := 0; i < everygroup; i++ {
		// 在第一轮分配的时候优先将节点分配到没有监督节点的分组中去
		for id, addr := range svnodes {
		reHash:
			key := id + addr + chash1.GetRandom()
			group := cHashRing.Get(key)
			ret := SaveNode(i, doc, id, addr, group.Id, true)
			if !ret {
				goto reHash
			}
		}
	}

	// for id, addr := range nodetable {
	// 	// 以节点的id,ip地址在加一个随机数计算chash
	// 	key := id + addr + chash1.GetRandom()
	// 	group := cHashRing.Get(key)
	// 	SaveNode(-1, doc, id, addr, group.Id, false)
	// }
	doc.WriteToFile(xmlpath)
	if !CheckIsDivided(doc) {
		// 删除文件 重新分组
		del := os.Remove(xmlpath)
		if del != nil {
			fmt.Println(del)
		}
		fmt.Println("节点分配不合理,某些组可能没有监督节点或者节点总数不够，无法进行Raft共识")
		goto Check
	}
	return doc
}

// 测试获取本地IP地址
func getlocalip() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, addr := range addrs {
		fmt.Println(addr.String())
	}
	fmt.Println("----------分割线----------")
	info, _ := net.InterfaceAddrs()
	for _, addr := range info {
		fmt.Println(strings.Split(addr.String(), "/")[0])
	}
}

func main() {
	if os.Args[1] == "1" {
		del := os.Remove(xmlpath)
		if del != nil {
			fmt.Println(del)
		}
		doc := NodeDistribution()
		if doc == nil {
			fmt.Println("err")
		}
	}
	if os.Args[1] == "2" {
		fmt.Println("测试监督节点暴露后的重选：")
		//影响最小的方式是直接把节点删了然后随机选取一个节点加进去
		doc := etree.NewDocument()
		if err := doc.ReadFromFile(xmlpath); err != nil {
			initxml(doc, xmlpath)
		}
		resetSvnode(doc, "N49")
	}

	//getlocalip()
	// node := "N14228228282"
	// fmt.Println(strings.Trim(node, "N"))
}

// {
// 	fmt.Println("测试新加入一个节点，对它进行分组")
// 	fmt.Println("请在下方输入要存入节点的信息，示例:N0-127.0.0.1:1234")
// 	stdReader := bufio.NewReader(os.Stdin)
// 	data, err := stdReader.ReadString('\n')
// 	if err != nil {
// 		fmt.Println("Error reading from stdin")
// 		panic(err)
// 	}
// 	data = strings.TrimSpace(data)
// 	slice := strings.Split(data, "-")
// 	if len(slice) != 2 {
// 		fmt.Println("输入不合法")
// 	}
// 	fmt.Println("你的输入是：", slice[0], slice[1])
// 	key := data + chash1.GetRandom()
// 	group := cHashRing.Get(key)
// 	SaveNode(doc, slice[0], slice[1], group.Id, false)
// 	doc.WriteToFile(xmlpath)
// 	fmt.Printf("节点分配完成，所处分组为：Group%d\n", group.Id)
// }

/********监督节点的暴露重选
//Todo 监督节点是怎么暴露的？
1，是只在一个组暴露还是多个组暴露
2， 在暴露的组内将它变成普通节点，在其他组内删除该节点
**************/

func resetSvnode(doc *etree.Document, id string) {
	var svNewnodeid, svNewnodeaddr string
	var newindex int
	var First int
	for i := 0; i < groupNum; i++ {
		Group := doc.FindElement("./Nodeinfo/Group" + strconv.Itoa(i))
		for j := 0; j < len(Group.ChildElements()); j++ {
			index := Group.SelectElement("Nodeindex" + strconv.Itoa(j+1))
			if nodeid := index.SelectAttr("NodeID"); nodeid != nil {
				if nodeid.Value == id { //不要轻易的删除
					//Group.RemoveChild(index)
					index.RemoveAttr("SvFlag")
					First++
					//Todo 如果i不是最先发现的下面会出现bug
					if First == 1 {
						// 在分组内重选监督节点
						newindex, svNewnodeid, svNewnodeaddr = reChooseSvnode(Group, i, len(Group.ChildElements())+1)
						n := Group.SelectElement("Nodeindex" + strconv.Itoa(newindex))
						n.CreateAttr("SvFlag", "true")
						doc.Indent(4)
					} else {
						index.SelectAttr("EnableUsed").Value = "false"
						n := Group.CreateElement("Nodeindex" + strconv.Itoa(len(Group.ChildElements())+1))
						n.CreateAttr("NodeID", svNewnodeid)
						n.CreateAttr("NodeAddr", svNewnodeaddr)
						n.CreateAttr("EnableUsed", "true")
						n.CreateAttr("SvFlag", "true")
						doc.Indent(4)
					}
					break
				}
			}
		}
	}
	doc.WriteToFile(xmlpath)
}

func reChooseSvnode(Group *etree.Element, i, n int) (int, string, string) {
	index := i
	// 重选的这个节点不能是监督节点，不能是无效节点,不能是原节点
	for {
		rand.Seed(time.Now().UnixNano())
		index = rand.Intn(n)
		if index == i {
			continue
		}
		in := Group.SelectElement("Nodeindex" + strconv.Itoa(index))
		if sv := in.SelectAttr("SvFlag"); sv != nil {
			continue
		}
		if in.SelectAttr("EnableUsed").Value == "false" {
			continue
		}
		break
	}

	id := Group.SelectElement("Nodeindex" + strconv.Itoa(index)).SelectAttr("NodeID").Value
	addr := Group.SelectElement("Nodeindex" + strconv.Itoa(index)).SelectAttr("NodeAddr").Value

	return index, id, addr
}
