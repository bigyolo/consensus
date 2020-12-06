package block

import (
	"bytes"
	"encoding/gob"
	"log"
)

//定义区块data域的数据结构
type Bdata struct {
	Msg       string //请求的内容
	Timestamp int64  // 每次客户端请求的时间戳
	// 其他结构可以继续扩充
}

// Block keeps block headers
type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

// Serialize serializes the block
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock deserializes a block
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

func DecodeBlockData(data []byte) interface{} {
	var ret []Bdata
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&ret)
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func EncodeBlockData(Bd interface{}) []byte {
	var r bytes.Buffer
	encoder := gob.NewEncoder(&r)
	err := encoder.Encode(Bd)
	if err != nil {
		log.Panic(err)
	}
	return r.Bytes()
}
