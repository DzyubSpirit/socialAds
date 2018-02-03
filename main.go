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
	"github.com/DzyubSpirit/firego"
	"io/ioutil"
	"golang.org/x/oauth2/google"
	"context"
)

const (
	dbFile         = "userInfo.dat"
	lastPostBucket = "lastPost"
	postCooldown   = 8 * time.Hour
	credsBytes     = "social-ads-1708a-firebase-adminsdk-qtbhe-1c11808167.json"
	databaseURL    = "https://social-ads-1708a.firebaseio.com/"
)

// NewFirebaseWithCreds loads credentials for firebase and connects to firebase
func NewFirebaseWithCreds(credsFilename string, databaseURL string) (*firego.Firebase, error) {
	d, err := ioutil.ReadFile(credsFilename)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(d, "https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/firebase.database")
	if err != nil {
		return nil, err
	}

	f := firego.New(databaseURL, conf.Client(context.Background()))
	return f, nil
}

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

	fb, err := NewFirebaseWithCreds(credsBytes, databaseURL)
	if err != nil {
		log.Fatalf("error creating firebase connection: %v", err)
	}
	usersRef := fb.Child("users")
	usersRef.ChildAdded(func(snapshot firego.DataSnapshot, previousChildKey string) {
		var user map[string]interface{}
		var ok bool
		if user, ok = snapshot.Value.(map[string]interface{}); !ok {
			log.Printf("error converting snapshot value to map")
			return
		}

		addr := cli.CreateWallet()
		user["address"] = addr
		err := usersRef.Update(user)
		if err != nil {
			log.Printf("error updating user with wallet address %q: %v", addr, err)
		}
	})

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
