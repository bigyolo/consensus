package chash1

//package main

// 利用一致性hash算法实现节点分组与监督节点的选择

//一致性哈希(Consistent Hashing)
//author: Xiong Chuan Liang
//date: 2015-2-20

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// 虚拟节点,解决数据倾斜的问题，每个节点分几个虚拟节点
/*
	虚拟节点的存在可以使hash环中的节点命中率变的均衡。
	虚拟节点越多，分布越均匀。
	但会带来数据牺牲，真实节点增加或者减少时
	由于虚拟节点数量剧烈变化，数据的重新分配可能会影响到更多的真实节点。
	因为有可能所有虚拟节点的下一个节点列表覆盖了其他所有真实节点。
	所以，如果key与服务无关，可以适当调大这个值，达到良好的均衡效果
	服务真实节点较多、数量变化频繁时，适当减少或者不设置，以减少数据迁移带来的影响，提高系统整体的可用性
*/
/*
 */

const DEFAULT_REPLICAS = 10000

type HashRing []uint32

func (c HashRing) Len() int {
	return len(c)
}

func (c HashRing) Less(i, j int) bool {
	return c[i] < c[j]
}

func (c HashRing) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type Node struct {
	Id        int
	Randomstr string
	// Ip       string
	// Port     int
	// HostName string
	Weight int
}

// NewNode 节点信息，IP地址 还有一个权重值
func NewNode(id int, rstr string, weight int) *Node {
	return &Node{
		Id:        id,
		Randomstr: rstr,
		// Ip:       ip,
		// Port:     port,
		// HostName: name,
		Weight: weight,
	}
}

type Consistent struct {
	Nodes     map[uint32]Node
	numReps   int
	Resources map[int]bool
	ring      HashRing
	sync.RWMutex
}

func NewConsistent() *Consistent {
	return &Consistent{
		Nodes:     make(map[uint32]Node), // 节点hash和节点IP地址
		numReps:   DEFAULT_REPLICAS,      // 解决数据倾斜的每个节点的副本数
		Resources: make(map[int]bool),
		ring:      HashRing{},
	}
}

func (c *Consistent) Add(node *Node) bool {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Resources[node.Id]; ok {
		return false
	}

	count := c.numReps * node.Weight // 根据权重值调整数据倾斜程度
	for i := 0; i < count; i++ {
		str := c.joinStr(i, node)
		c.Nodes[c.hashStr(str)] = *(node)
	}
	c.Resources[node.Id] = true
	c.sortHashRing()
	return true
}

func (c *Consistent) sortHashRing() {
	c.ring = HashRing{}
	for k := range c.Nodes {
		c.ring = append(c.ring, k)
	}
	sort.Sort(c.ring)
}

func (c *Consistent) joinStr(i int, node *Node) string {
	return "Group" + strconv.Itoa(i) + strconv.Itoa(node.Weight) +
		"-" + strconv.Itoa(i) +
		"-" + strconv.Itoa(node.Id)
}

// MurMurHash算法 :https://github.com/spaolacci/murmur3
func (c *Consistent) hashStr(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

// 数据分配，根据key计算出的hash值确定 这个key由哪个节点承载
func (c *Consistent) Get(key string) Node {
	c.RLock()
	defer c.RUnlock()

	hash := c.hashStr(key)
	fmt.Println("hash", hash)
	i := c.search(hash)

	return c.Nodes[c.ring[i]]
}

func (c *Consistent) search(hash uint32) int {

	i := sort.Search(len(c.ring), func(i int) bool { return c.ring[i] >= hash })
	if i < len(c.ring) {
		if i == len(c.ring)-1 {
			return 0
		} else {
			return i
		}
	} else {
		return len(c.ring) - 1
	}
}

func (c *Consistent) Remove(node *Node) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Resources[node.Id]; !ok {
		return
	}

	delete(c.Resources, node.Id)

	count := c.numReps * node.Weight
	for i := 0; i < count; i++ {
		str := c.joinStr(i, node)
		delete(c.Nodes, c.hashStr(str))
	}
	c.sortHashRing()
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

// func tets() {
// 	//初始化一个空hash环
// 	cHashRing := NewConsistent()

// 	for i := 0; i < 4; i++ {
// 		node := NewNode(i, getRandom(), 1)
// 		cHashRing.Add(node)
// 	}
// 	// hash环已经分好了

// 	for k, v := range cHashRing.Nodes {
// 		fmt.Println("Hash:", k, " ID:Group", v.Id)
// 	}
// 	t0 := time.Now()
// 	for i := 0; i < 10; i++ {

// 		s := getRandom()

// 		k := cHashRing.Get(s)

// 		fmt.Printf("S%d 数据属于%d\n", i, k.Id)

// 	}
// 	t1 := time.Now()
// 	fmt.Println("耗时：", t1.Sub(t0))

// 	ipMap := make(map[string]int, 0)
// 	for i := 0; i < 1000; i++ {
// 		si := fmt.Sprintf("key%d", i)
// 		k := cHashRing.Get(si)
// 		if _, ok := ipMap[k.Ip]; ok {
// 			ipMap[k.Ip]++
// 		} else {
// 			ipMap[k.Ip] = 1
// 		}
// 	}

// 	for k, v := range ipMap {
// 		fmt.Println("Node IP:", k, " count:", v)
// 	}

// }
// func main() {
// 	tets()
// }

/*
分布:
Node IP: 172.18.1.2  count: 115
Node IP: 172.18.1.8  count: 111
Node IP: 172.18.1.3  count: 94
Node IP: 172.18.1.1  count: 84
Node IP: 172.18.1.7  count: 107
Node IP: 172.18.1.6  count: 117
Node IP: 172.18.1.4  count: 92
Node IP: 172.18.1.5  count: 112
Node IP: 172.18.1.0  count: 63
Node IP: 172.18.1.9  count: 105
*/
