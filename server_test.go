package zenrpc_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"
)

var rpc = zenrpc.NewServer(zenrpc.Options{BatchMaxLen: 5, AllowCORS: true})

func init() {
	rpc.Register("arith", &testdata.ArithService{})
	rpc.Register("", &testdata.ArithService{})
	//rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))
}

func TestServer_SMD(t *testing.T) {
	r := rpc.SMD()
	if b, err := json.Marshal(r); err != nil {
		t.Fatal(err)
	} else if !bytes.Contains(b, []byte("default")) {
		t.Error(string(b))
	}
}

func TestServer_SmdGenerate(t *testing.T) {
	rpc := zenrpc.NewServer(zenrpc.Options{})
	rpc.Register("phonebook", testdata.PhoneBook{DB: testdata.People})
	rpc.Register("arith", testdata.ArithService{})
	rpc.Register("printer", testdata.PrintService{})
	rpc.Register("", testdata.ArithService{})

	b, _ := json.MarshalIndent(rpc.SMD(), "", "  ")

	testData, err := ioutil.ReadFile("./testdata/testdata/arithsrv-smd.json")
	if err != nil {
		t.Fatalf("open test data file")
	}

	if !bytes.Equal(b, testData) {
		t.Fatalf("bad zenrpc output")
	}
}
