package main

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha1"
	"fmt"
	"math/big"
	"math/rand"
	"time"
)

const (
	bucketSize = 160
	peerCount  = 100
	keyCount   = 200
)

type Peer struct {
	id   *big.Int
	dht  []*Peer
	data map[string][]byte
}

func NewPeer(id *big.Int) *Peer {
	return &Peer{
		id:   id,
		dht:  make([]*Peer, bucketSize),
		data: make(map[string][]byte),
	}
}

func (p *Peer) SetValue(key, value []byte) bool {
	hash := sha1.Sum(key)
	hashInt := new(big.Int).SetBytes(hash[:])
	id := hashInt.Bytes()[:bucketSize/8]
	if bytes.Compare(id, p.id.Bytes()) != 0 {
		return false
	}
	if _, ok := p.data[string(key)]; ok {
		return true
	}
	p.data[string(key)] = value
	keyHash := sha1.Sum(key)
	bucketIndex := getBucketIndex(p.id, keyHash[:])
	for i := 0; i < bucketSize/8; i++ {
		if p.dht[bucketIndex] != nil {
			p.dht[bucketIndex].SetValue(key, value)
		}
		bucketIndex = (bucketIndex + 1) % bucketSize
	}
	return true
}

func (p *Peer) GetValue(key []byte) []byte {
	if value, ok := p.data[string(key)]; ok {
		return value
	}
	keyHash := sha1.Sum(key)
	var closestPeers []*Peer
	for _, peer := range p.dht {
		if peer != nil {
			closestPeers = append(closestPeers, peer)
		}
	}
	closestPeers = getClosestPeers(closestPeers, keyHash[:])
	for _, peer := range closestPeers {
		if value, ok := peer.data[string(key)]; ok {
			keyValue := sha1.Sum(value)
			if bytes.Equal(keyValue[:bucketSize/8], keyHash[:bucketSize/8]) {
				return value
			}
		}
	}
	return nil
}

func getBucketIndex(id *big.Int, targetID []byte) int {
	distance := new(big.Int).Xor(id, new(big.Int).SetBytes(targetID))
	return bucketSize - distance.BitLen() - 1
}

func getClosestPeers(peers []*Peer, targetID []byte) []*Peer {
	var closestPeers []*Peer
	for i := 0; i < 2 && len(peers) > 0; i++ {
		closestPeerIndex := 0
		closestPeerDistance := new(big.Int).Xor(peers[0].id, new(big.Int).SetBytes(targetID))
		for j, peer := range peers {
			distance := new(big.Int).Xor(peer.id, new(big.Int).SetBytes(targetID))
			if distance.Cmp(closestPeerDistance) < 0 {
				closestPeerIndex = j
				closestPeerDistance = distance
			}
		}
		closestPeers = append(closestPeers, peers[closestPeerIndex])
		peers = append(peers[:closestPeerIndex], peers[closestPeerIndex+1:]...)
	}
	return closestPeers
}

func main() {
	rand.Seed(time.Now().UnixNano())

	//初始化peers
	peers := make([]*Peer, peerCount)
	for i := 0; i < peerCount; i++ {
		id := randInt()
		peers[i] = NewPeer(id)
		for j := 0; j < bucketSize; j++ {
			if rand.Intn(1) == 0 {
				peers[i].dht[j] = peers[rand.Intn(peerCount)]
			}
		}
	}

	// Set values
	keys := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		key := randBytes()
		value := randBytes()
		keys[i] = key
		peerIndex := rand.Intn(peerCount)
		peers[peerIndex].SetValue(key, value)
	}

	// Get values
	for i := 0; i < keyCount/2; i++ {
		key := keys[rand.Intn(keyCount)]
		peerIndex := rand.Intn(peerCount)
		value := peers[peerIndex].GetValue(key)
		fmt.Println("peer is", string(key))
		if value != nil {
			fmt.Printf("value is %s", string(value))
		} else {
			fmt.Printf("Can't find value")
		}
	}
}

func randInt() *big.Int {
	max := new(big.Int).Lsh(big.NewInt(1), bucketSize)
	n, err := crand.Int(crand.Reader, max)
	if err != nil {
		panic(err)
	}
	return n
}

func randBytes() []byte {
	b := make([]byte, rand.Intn(100)+1)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}
