package models

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	//	"github.com/davecgh/go-spew/spew"

	_ "github.com/mattn/go-sqlite3"
)

type MoneroPrice struct {
	Monero struct {
		Usd float64 `json:"usd"`
	} `json:"monero"`
}

type MoneroRepository struct {
	db           *sql.DB
	xmrTransfers []MoneroPrice
	xmrWallets   [][]int
}

func NewMoneroRepository(db *sql.DB) *MoneroRepository {
	return &MoneroRepository{
		db:           db,
		xmrTransfers: []MoneroPrice{},
		xmrWallets:   [][]int{},
	}
}

func (mr *MoneroRepository) getBalance(checkID string, userID int) (float64, error) {
	portID := getPortID(mr.xmrWallets, userID)

	found := true
	if portID == -100 {
		found = false
	}

	if found {
		fmt.Println("Port ID for user", userID, "is", portID)
	} else {
		fmt.Println("Port ID not found for user", userID)
	}

	url := "http://localhost:" + strconv.Itoa(portID) + "/json_rpc"

	payload := struct {
		Jsonrpc string `json:"jsonrpc"`
		Id      int    `json:"id"`
		Method  string `json:"method"`
		Params  struct {
			PaymentID string `json:"payment_id"`
		} `json:"params"`
	}{
		Jsonrpc: "2.0",
		Id:      0,
		Method:  "get_payments",
		Params: struct {
			PaymentID string `json:"payment_id"`
		}{
			PaymentID: checkID,
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return 0.0, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return 0.0, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return 0.0, err
	}

	fmt.Println(result)

	resultMap, ok := result["result"].(map[string]interface{})
	if !ok {
		return 0.0, fmt.Errorf("result key not found in response")
	}

	payments, ok := resultMap["payments"].([]interface{})
	if !ok {
		return 0.0, fmt.Errorf("payments key not found in result map")
	}

	if len(payments) == 0 {
		return 0.0, fmt.Errorf("no payments found for payment ID %s", checkID)
	}

	amount := payments[0].(map[string]interface{})["amount"].(float64)

	return amount / math.Pow(10, 12), nil
}

func getPortID(xmrWallets [][]int, userID int) int {
	for _, innerList := range xmrWallets {
		if innerList[0] == userID {
			return innerList[1]
		}
	}
	return -100
}
