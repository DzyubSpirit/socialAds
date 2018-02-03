package main

import (
	"github.com/dzyubspirit/blockchain_go"
	"net/http"
	"log"
	"flag"
	"encoding/json"
	"strconv"
	"time"
	"github.com/boltdb/bolt"
	"fmt"
)

const (
	dbFile         = "userInfo.dat"
	lastPostBucket = "lastPost"
	postCooldown   = 8 * time.Hour
)

func main() {
	sAddr := flag.String("main_address", "", "a wallet of the service")
	flag.Parse()

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatalf("bolt.Open(%q, 0600, nil): %v", dbFile, err)
	}

	cli := blockchain.CLI{}
	if *sAddr == "" {
		*sAddr = cli.CreateWallet()
	}
	log.Printf("main addr: %q", *sAddr)
	cli.CreateBlockchain(*sAddr)

	http.HandleFunc("/reward", func(res http.ResponseWriter, req *http.Request) {
		p := req.URL.Query()
		log.Println(p)
		addr := p["address"][0]
		amount, err := strconv.Atoi(p["amount"][0])
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte("amount must be integer"))
			return
		}

		date := time.Now()
		canBeInserted := true
		err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(lastPostBucket))
			if err != nil {
				return fmt.Errorf("tx.CreateBucketIfNotExists(): %v", err)
			}

			lastPost := b.Get([]byte(addr))
			if lastPost != nil {
				var oldDate time.Time
				err = oldDate.UnmarshalBinary(lastPost)
				if err != nil {
					return fmt.Errorf("error unmarshaling date %s: %v", lastPost, err)
				}

				if oldDate.Add(postCooldown).After(date) {
					canBeInserted = false
					return nil
				}
			}

			dateBytes, err := date.MarshalBinary()
			if err != nil {
				return fmt.Errorf("error marshaling date %v: %v", date, err)
			}
			err = b.Put([]byte(addr), dateBytes)
			if err != nil {
				return fmt.Errorf("error putting last date in bolt: %v", err)
			}

			return nil
		})
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Printf("error in transaction: %v", err)
			return
		}

		if canBeInserted {
			cli.Send(*sAddr, addr, amount)
		}

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
	log.Fatal(http.ListenAndServe(":8080", nil))
}
