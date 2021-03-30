package testdata

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/vmkteam/zenrpc/v2"
)

func TestSmdGenerate(t *testing.T) {
	rpc := zenrpc.NewServer(zenrpc.Options{})
	rpc.Register("phonebook", PhoneBook{DB: People})
	rpc.Register("arith", ArithService{})
	rpc.Register("printer", PrintService{})
	rpc.Register("", ArithService{})

	b, _ := json.Marshal(rpc.SMD())

	testData, err := ioutil.ReadFile("./testdata/arithsrv-smd.json")
	if err != nil {
		t.Fatalf("open test data file")
	}

	if !bytes.Equal(b, testData) {
		t.Fatalf("bad zenrpc output")
	}
}
