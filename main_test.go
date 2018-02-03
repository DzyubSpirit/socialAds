package main

import (
	"testing"
	"net/http"
	"encoding/json"
)

const (
	host = "http://localhost:8080"
)

func TestBalance(t *testing.T) {
	res, err := http.Get(host + "/createWallet")
	if err != nil {
		t.Errorf("http.Get(\"http://localhost:8080/createWallet\"): %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("http.Get(\"http://localhost:8080/createWallet\"), response code: %s", res.StatusCode)
	}

	var addr string
	err = json.NewDecoder(res.Body).Decode(&addr)
	if err != nil {
		t.Errorf("error decoding response for creating new Wallet: %v", err)
	}

	r, err := http.NewRequest("GET", host+"/reward", nil)
	if err != nil {
		t.Errorf("error creating request for reward: %v", err)
	}

	vals := r.URL.Query()
	vals.Add("amount", "11")
	vals.Add("address", addr)
	r.URL.RawQuery = vals.Encode()

	res, err = http.DefaultClient.Do(r)
	if err != nil || res.StatusCode != 200 {
		t.Errorf("error doing request for reward: err: %v", err)
	}

	r, err = http.NewRequest("GET", host+"/balance", nil)
	if err != nil {
		t.Errorf("error creating request for balance: %v", err)
	}

	vals = r.URL.Query()
	vals.Add("address", addr)
	r.URL.RawQuery = vals.Encode()

	res, err = http.DefaultClient.Do(r)
	if err != nil && res.StatusCode != 200 {
		t.Errorf("error doing request for balance,  err: %v", err)
	}

	var balance int
	err = json.NewDecoder(res.Body).Decode(&balance)
	if err != nil {
		t.Errorf("error doing request for balance: %v", err)
	}

	if balance != 11 {
		t.Errorf("balance should be 11, actual: %v", balance)
	}
}
