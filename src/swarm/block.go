package swarm

import (
	"encoding/json"
	"log"
)

type Block struct {
	Id         string
	Blockchain string
	Compiler   string

	Heartbeats map[string]Heartbeat

	//Mapping of hosts -> what they store
	StorageMapping map[string]interface{}
}

func (b *Block) SwarmId() string {
	return b.Blockchain
}

func (b *Block) BlockId() string {
	return b.Id
}

func (s *StateSteady) IntegrateBlock(b Block) {
	// verify proof-of-storage
	// determine ordering, sort heartbeats by ordering (producing a slice of heartbeats)
	// generate entropy

	// pull transactions out of heartbeats in order
	// process each individually, checking to see if it's valid or not
}

func (s *StateSteady) verify(b Block) {
	// verifies that a block is valid
	// checks timestamps, checks the sender, etc.
}

func (b *Block) MarshalString() string {
	o, err := json.Marshal(b)
	if err != nil {
		log.Fatal("Unable to marshal Block:", err)
	}
	return string(o)
}

func UnmarshalBlock(encoded string) (b *Block, err error) {
	b = new(Block)
	err = json.Unmarshal([]byte(encoded), b)
	return b, err
}
