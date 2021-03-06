package quorum

import (
	"bytes"
	"common"
	"common/crypto"
	"common/log"
	"crypto/ecdsa"
	"encoding/gob"
	"fmt"
	"sync"
)

// Identifies other members of the quorum
type Sibling struct {
	index     byte // not sure that this is the appropriate place for this variable
	address   common.Address
	publicKey *crypto.PublicKey
}

type quorum struct {
	// Network Variables
	siblings     [common.QuorumSize]*Sibling // list of all siblings in quorum
	siblingsLock sync.RWMutex                // prevents race conditions
	// meta quorum

	// file proofs stage 2

	// Compile Variables
	currentEntropy  common.Entropy // Used to generate random numbers during compilation
	upcomingEntropy common.Entropy // Used to compute entropy for next block

	// Batch management
	parent *batchNode
}

// returns the current state of the quorum in human-readable form
func (q *quorum) Status() (b string) {
	b = "\tSiblings:\n"
	for _, s := range q.siblings {
		if s != nil {
			b += fmt.Sprintf("\t\t%v %v\n", s.index, s.address)
		}
	}
	return
}

// Convert quorum to []byte
// Only the siblings and entropy are encoded.
func (q *quorum) GobEncode() (gobQuorum []byte, err error) {
	// if q == nil, encode a zero quorum
	if q == nil {
		q = new(quorum)
	}

	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)

	// only encode non-nil siblings
	var encSiblings []*Sibling
	for _, s := range q.siblings {
		if s != nil {
			encSiblings = append(encSiblings, s)
		}
	}
	err = encoder.Encode(encSiblings)
	if err != nil {
		return
	}
	err = encoder.Encode(q.currentEntropy)
	if err != nil {
		return
	}
	err = encoder.Encode(q.upcomingEntropy)
	if err != nil {
		return
	}

	gobQuorum = w.Bytes()
	return
}

// Convert []byte to quorum
// Only the siblings and entropy are decoded.
func (q *quorum) GobDecode(gobQuorum []byte) (err error) {
	// if q == nil, make a new quorum and decode into that
	if q == nil {
		q = new(quorum)
	}

	r := bytes.NewBuffer(gobQuorum)
	decoder := gob.NewDecoder(r)

	// not all siblings were encoded
	var encSiblings []*Sibling
	err = decoder.Decode(&encSiblings)
	if err != nil {
		return
	}
	for _, s := range encSiblings {
		q.siblings[s.index] = s
	}
	err = decoder.Decode(&q.currentEntropy)
	if err != nil {
		return
	}
	err = decoder.Decode(&q.upcomingEntropy)
	if err != nil {
		return
	}
	return
}

// Returns true if the values of the siblings are equivalent
func (s0 *Sibling) compare(s1 *Sibling) bool {
	// false if either sibling is nil
	if s0 == nil || s1 == nil {
		return false
	}

	// return false if the addresses are not equal
	if s0.address != s1.address {
		return false
	}

	// return false if the public keys are not equivalent
	compare := s0.publicKey.Compare(s1.publicKey)
	if compare != true {
		return false
	}

	return true
}

// siblings are processed in a random order each block, determined by the
// entropy for the block. siblingOrdering() deterministically picks that
// order, using entropy from the state.
func (q *quorum) siblingOrdering() (siblingOrdering []byte) {
	// create an in-order list of siblings
	for i, s := range q.siblings {
		if s != nil {
			siblingOrdering = append(siblingOrdering, byte(i))
		}
	}

	// shuffle the list of siblings
	for i := range siblingOrdering {
		newIndex, err := q.randInt(i, len(siblingOrdering))
		if err != nil {
			log.Fatalln(err)
		}
		tmp := siblingOrdering[newIndex]
		siblingOrdering[newIndex] = siblingOrdering[i]
		siblingOrdering[i] = tmp
	}

	return
}

func (s *Sibling) GobEncode() (gobSibling []byte, err error) {
	// Error checking for nil values
	if s == nil {
		err = fmt.Errorf("Cannot encode nil sibling")
		return
	}
	if s.publicKey == nil {
		err = fmt.Errorf("Cannot encode nil value s.publicKey")
		return
	}
	epk := (*ecdsa.PublicKey)(s.publicKey)
	if epk.X == nil {
		err = fmt.Errorf("Cannot encode nil value s.publicKey.X")
		return
	}
	if epk.Y == nil {
		err = fmt.Errorf("Cannot encode nil value s.publicKey.Y")
		return
	}

	// Encoding the sibling
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err = encoder.Encode(s.index)
	if err != nil {
		return
	}
	err = encoder.Encode(s.address)
	if err != nil {
		return
	}
	err = encoder.Encode(s.publicKey)
	if err != nil {
		return
	}
	gobSibling = w.Bytes()
	return
}

func (s *Sibling) GobDecode(gobSibling []byte) (err error) {
	if s == nil {
		err = fmt.Errorf("Cannot decode into nil sibling")
		return
	}

	r := bytes.NewBuffer(gobSibling)
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&s.index)
	if err != nil {
		return
	}
	err = decoder.Decode(&s.address)
	if err != nil {
		return
	}
	err = decoder.Decode(&s.publicKey)
	if err != nil {
		return
	}
	return
}

// Update the state according to the information presented in the heartbeat
func (q *quorum) processHeartbeat(hb *heartbeat) (newSiblings []*Sibling, err error) {
	// add hopefuls to any available slots
	// q.siblings has already been locked by compile()
	for _, s := range hb.hopefuls {
		j := 0
		for {
			if j == common.QuorumSize {
				log.Infoln("failed to add hopeful: quorum already full")
				break
			}
			if q.siblings[j] == nil {
				println("placed hopeful at index", j)
				s.index = byte(j)
				q.siblings[s.index] = s
				newSiblings = append(newSiblings, s)
				break
			}
			j++
		}
	}

	// Add the entropy to UpcomingEntropy
	th, err := crypto.CalculateTruncatedHash(append(q.upcomingEntropy[:], hb.entropy[:]...))
	q.upcomingEntropy = common.Entropy(th)

	return
}

// Use the entropy stored in the state to generate a random integer [low, high)
// randInt only runs during compile(), when the mutexes are already locked
func (q *quorum) randInt(low int, high int) (randInt int, err error) {
	// verify there's a gap between the numbers
	if low == high {
		err = fmt.Errorf("low and high cannot be the same number")
		return
	}

	// Convert CurrentEntropy into an int
	rollingInt := 0
	for i := 0; i < 4; i++ {
		rollingInt = rollingInt << 8
		rollingInt += int(q.currentEntropy[i])
	}

	randInt = (rollingInt % (high - low)) + low

	// Convert random number seed to next value
	truncatedHash, err := crypto.CalculateTruncatedHash(q.currentEntropy[:])
	q.currentEntropy = common.Entropy(truncatedHash)
	return
}
