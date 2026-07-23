package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	addr := flag.String("addr", "localhost:9999", "listen address")
	withSRI := flag.Bool("sri", false, "add SRI hashes to the external scripts")
	flag.Parse()

	const phonebook = "phonebook"

	rpc := zenrpc.NewServer(zenrpc.Options{
		ExposeSMD:              true,
		AllowCORS:              true,
		DisableTransportChecks: true,
	})
	rpc.Register(phonebook, testdata.PhoneBook{DB: testdata.People})
	rpc.Register("arith", testdata.ArithService{})
	rpc.Register("printer", testdata.PrintService{})
	rpc.Register("", testdata.ArithService{}) // public

	rpc.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))
	rpc.Use(zenrpc.Metrics(""), testdata.SerialPeopleAccess(phonebook))

	rpc.SetLogger(log.New(os.Stderr, "A", log.LstdFlags))

	http.Handle("/", rpc)
	http.HandleFunc("/ws", rpc.ServeWS)
	http.Handle("/metrics", promhttp.Handler())

	var docHandler http.HandlerFunc
	if *withSRI {
		docHandler = zenrpc.SMDBoxHandlerWithSRI
	} else {
		docHandler = zenrpc.SMDBoxHandler
	}
	http.HandleFunc("/doc", docHandler)

	log.Printf("starting arithsrv on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
