package swarm

import (
	"testing"
)

func TestIntegrate(t *testing.T) {

}

func Testverify(t *testing.T) {

}

/* As block marshaling got nuked, this shouldn't be needed
func TestMarshaling(t *testing.T) {
	b := new(Block)
	b.Id = "2"
	b.EntropyStage1 = make(map[string]string)
	b.EntropyStage2 = make(map[string]string)
	b.StorageMapping = make(map[string]interface{})

	s := b.MarshalString()

	b2, err := UnmarshalBlock(s)
	if err != nil {
		t.Fatal(err)
	}

	if b.Id != b2.Id {
		t.Fatal("Id not equal")
	}
}*/
