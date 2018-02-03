package main

import (
	"github.com/dzyubspirit/blockchain_go"
	"net/http"
	"log"
	"flag"
	"encoding/json"
	"strconv"
)

const multiplier = 1.2

func main() {
	sAddr := flag.String("main_address", "", "a wallet of the service")
	flag.Parse()

	cli := blockchain.CLI{}
	if *sAddr == "" {
		*sAddr = cli.CreateWallet()
	}
	log.Printf("main addr: %q", *sAddr)
	cli.CreateBlockchain(*sAddr)

	http.HandleFunc("/reward", func(res http.ResponseWriter, req *http.Request) {
		p := req.URL.Query()
		addr := p["address"][0]
		friendsCount, err := strconv.Atoi(p["friendsCount"][0])
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte("friendsCount must be integer"))
			return
		}

		cli.Send(*sAddr, addr, int(float64(friendsCount)*multiplier))
		res.Write([]byte("Okey"))
	})

	http.HandleFunc("/balance", func(res http.ResponseWriter, req *http.Request) {
		addr := req.URL.Query()["address"][0]
		balance := cli.GetBalance(addr)
		err := json.NewEncoder(res).Encode(balance)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/createWallet", func(res http.ResponseWriter, req *http.Request) {
		addr := cli.CreateWallet()
		err := json.NewEncoder(res).Encode(addr)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Printf("json.Encode(%q): %v", addr, err)
		}
	})
	http.ListenAndServe(":80", nil)
}
