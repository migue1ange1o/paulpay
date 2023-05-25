package models

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	//	"github.com/davecgh/go-spew/spew"

	"github.com/gabstv/go-monero/walletrpc"
	_ "github.com/mattn/go-sqlite3"
)

type RPCResponse struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		IntegratedAddress string `json:"integrated_address"`
		PaymentID         string `json:"payment_id"`
	} `json:"result"`
}

type MoneroPrice struct {
	Monero struct {
		Usd float64 `json:"usd"`
	} `json:"monero"`
}

type MoneroRepositoryInterface interface {
	getBalance(checkID string, userID int) (float64, error)
	StartMoneroWallet(portInt, userID int, user User)
	GetPortID(xmrWallets [][]int, userID int) int
	CheckMoneroPort(userID int) bool
	GetNewAccountXMR() (string, string)
	StopMoneroWallet(user User)
}

type MoneroRepository struct {
	Db           *sql.DB
	UserRepo     *UserRepository
	XmrTransfers []MoneroPrice
	XmrWallets   [][]int
}

func NewMoneroRepository(db *sql.DB, ur *UserRepository) *MoneroRepository {
	return &MoneroRepository{
		Db:           db,
		UserRepo:     ur,
		XmrTransfers: []MoneroPrice{},
		XmrWallets:   [][]int{},
	}
}

func (mr *MoneroRepository) getBalance(checkID string, userID int) (float64, error) {
	portID := mr.GetPortID(mr.XmrWallets, userID)

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

func (mr *MoneroRepository) StartMoneroWallet(portInt, userID int, user User) {
	portID := mr.GetPortID(mr.XmrWallets, userID)
	found := true

	if portID == -100 {
		found = false
	}

	portStr := strconv.Itoa(portID)

	if found {
		fmt.Println("Port ID for user", userID, "is", portID)
	} else {
		fmt.Println("Port ID not found for user", userID)
		portStr = strconv.Itoa(portInt)
	}

	cmd := exec.Command("./start_xmr_wallet.sh", portStr, strconv.Itoa(userID))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Println("Error running command:", err)
		user.WalletUploaded = false
		user.WalletPending = false
		mr.UserRepo.Update(user)
		return
	}

	_ = walletrpc.New(walletrpc.Config{
		Address: "http://127.0.0.1:" + strconv.Itoa(portInt) + "/json_rpc",
	})

	fmt.Println("Done starting monero wallet for", portStr, userID)
	user.WalletUploaded = true
	user.WalletPending = true
	mr.UserRepo.Update(user)
}
func (mr *MoneroRepository) GetPortID(xmrWallets [][]int, userID int) int {
	for _, innerList := range xmrWallets {
		if innerList[0] == userID {
			return innerList[1]
		}
	}
	return -100
}

func (mr *MoneroRepository) CheckMoneroPort(userID int) bool {
	payload := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"make_integrated_address"}`)
	portID := mr.GetPortID(mr.XmrWallets, userID)

	found := true
	if portID == -100 {
		return false
	}

	if found {
		fmt.Println("Port ID for user", userID, "is", portID)
	} else {
		fmt.Println("Port ID not found for user", userID)
	}

	rpcURL_ := "http://127.0.0.1:" + strconv.Itoa(portID) + "/json_rpc"

	req, err := http.NewRequest("POST", rpcURL_, payload)
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	resp := &RPCResponse{}
	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		return false
	}

	return true
}

func (mr *MoneroRepository) GetNewAccountXMR() (string, string) {
	payload := strings.NewReader(`{"jsonrpc":"2.0","id":"0","method":"make_integrated_address"}`)
	userID := 1
	portID := mr.GetPortID(mr.XmrWallets, userID)

	found := true
	if portID == -100 {
		found = false
	}

	if found {
		fmt.Println("Port ID for user", userID, "is", portID)
	} else {
		fmt.Println("Port ID not found for user", userID)
	}

	rpcURL_ := "http://127.0.0.1:" + strconv.Itoa(portID) + "/json_rpc"

	req, err := http.NewRequest("POST", rpcURL_, payload)
	if err != nil {
		log.Println("ERROR CREATING req:", err)
		return "", ""
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR SENDING REQUEST:", err)
		return "", ""
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Println("ERROR: Non-200 response code received:", res.StatusCode)
		return "", ""
	}

	resp := &RPCResponse{}
	if err := json.NewDecoder(res.Body).Decode(resp); err != nil {
		log.Println("ERROR DECODING RESPONSE:", err)
		return "", ""
	}

	PayID := html.EscapeString(resp.Result.PaymentID)

	PayAddress := html.EscapeString(resp.Result.IntegratedAddress)

	log.Println("RETURNING XMR PAYID:", PayID)
	return PayID, PayAddress
}

func (mr *MoneroRepository) StopMoneroWallet(user User) {
	portID := mr.GetPortID(mr.XmrWallets, user.UserID)

	found := true
	if portID == -100 {
		found = false
	}

	if !found {
		fmt.Println("Port ID not found for user", user.UserID)
		return
	}

	portStr := strconv.Itoa(portID)

	cmd := exec.Command("monero/monero-wallet-rpc", "--rpc-bind-port", portStr, "--command", "stop_wallet")

	// Capture the output of the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running command: %v\n", err)
		return
	}

	// Print the output of the command
	fmt.Println(string(output))
}
