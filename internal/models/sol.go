package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	//"github.com/davecgh/go-spew/spew"
	//bin "github.com/gagliardetto/binary"
	"bytes"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"net/http"
	"time"

	"github.com/portto/solana-go-sdk/client"
	"github.com/shopspring/decimal"
)

type TransactionResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  struct {
		BlockTime int64 `json:"blockTime"`
		Meta      struct {
			ComputeUnitsConsumed int64         `json:"computeUnitsConsumed"`
			Err                  interface{}   `json:"err"`
			Fee                  int64         `json:"fee"`
			InnerInstructions    []interface{} `json:"innerInstructions"`
			LoadedAddresses      struct {
				Readonly []interface{} `json:"readonly"`
				Writable []interface{} `json:"writable"`
			} `json:"loadedAddresses"`
			LogMessages       []string      `json:"logMessages"`
			PostBalances      []int64       `json:"postBalances"`
			PostTokenBalances []interface{} `json:"postTokenBalances"`
			PreBalances       []int64       `json:"preBalances"`
			PreTokenBalances  []interface{} `json:"preTokenBalances"`
			Rewards           []interface{} `json:"rewards"`
			Status            struct {
				Ok interface{} `json:"Ok"`
			} `json:"status"`
		} `json:"meta"`
		Slot        int64 `json:"slot"`
		Transaction struct {
			Message struct {
				AccountKeys []string `json:"accountKeys"`
				Header      struct {
					NumReadonlySignedAccounts   int64 `json:"numReadonlySignedAccounts"`
					NumReadonlyUnsignedAccounts int64 `json:"numReadonlyUnsignedAccounts"`
					NumRequiredSignatures       int64 `json:"numRequiredSignatures"`
				} `json:"header"`
				Instructions []struct {
					Accounts       []int64 `json:"accounts"`
					Data           string  `json:"data"`
					ProgramIDIndex int64   `json:"programIdIndex"`
				} `json:"instructions"`
				RecentBlockhash string `json:"recentBlockhash"`
			} `json:"message"`
			Signatures []string `json:"signatures"`
		} `json:"transaction"`
	} `json:"result"`
	ID int64 `json:"id"`
}

type SuperChat struct {
	Name         string
	Message      string
	Address      string
	MediaURL     string
	AmountNeeded float64
	Completed    bool
	CreatedAt    string
	CheckedAt    string
	CryptoCode   string
	EncryptedIP  string
}

// Create a struct to represent the data
type Transaction struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Amount    int64  `json:"amount"`
}

// Create a struct to represent the data
type SolWallet struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
}

type SolRepositoryInterface interface {
	StartMonitoringSolana()
	CheckTransactionSolana(amt string, addr string, max_depth int) bool
	SetSolWallets(sW map[int]SolWallet)
	GetTransactionsForAddresses()
	GetTransactionsForAddressesFirst()
	AddSolanaTransactionStart(addr, sig string)
	AddSolanaTransaction(addr, sig string, amount int64)
	CreatePendingSolDono(name string, message string, mediaURL string, amountNeeded float64) SuperChat
	ContainsTransaction(sig string) bool
	CheckSameBalanceSol(wallet SolWallet) (SolWallet, bool)
	GetSOLBalance(address string) (float64, error)
	GetTransactionAmount(sig, addr string) (int64, bool)
	PrintSolTx(fromAddr, checkAddr, toAddr string, amountSent int64, sig string)
}

type SolRepository struct {
	db      *sql.DB
	wallets map[int]SolWallet
}

func NewSolRepository(db *sql.DB) *SolRepository {
	return &SolRepository{
		db:      db,
		wallets: map[int]SolWallet{},
	}
}

// Define a slice of Transaction objects
var transactions []Transaction
var solWallets = map[int]SolWallet{}

var firstRun bool = true

// Mainnet
var solClient = client.NewClient("https://api.mainnet-beta.solana.com")

func (sr *SolRepository) StartMonitoringSolana() {
	for {
		sr.GetTransactionsForAddresses()
	}
}

func (sr *SolRepository) CheckTransactionSolana(amt string, addr string, max_depth int) bool {
	decAmountReceived, _ := decimal.NewFromString(amt)
	decMultiplier := decimal.NewFromFloat(1000000000)
	result := decAmountReceived.Mul(decMultiplier)
	amountSent := result.IntPart()

	fmt.Println("Checking", addr, "for", amountSent, "lamport")

	startIndex := len(transactions) - max_depth // Calculate max depth of transactions to search
	if startIndex < 0 {
		startIndex = 0 // Make sure start index is not negative
	}

	for i := startIndex; i < len(transactions); i++ {
		transaction := transactions[i]
		if transaction.Address == addr && transaction.Amount == amountSent {
			return true
		}
	}
	return false
}

func (sr *SolRepository) SetSolWallets(sW map[int]SolWallet) {
	solWallets = sW
}

func (sr *SolRepository) GetTransactionsForAddresses() {
	for _, wallet := range solWallets {
		sameBalance := false

		wallet, sameBalance = sr.CheckSameBalanceSol(wallet)

		if sameBalance {
			fmt.Println("SOL:", wallet.Address[:7]+".. no new txs.")
			time.Sleep(10 * time.Second)
		} else {
			endpoint := rpc.MainNetBeta_RPC
			client := rpc.New(endpoint)
			out, err := client.GetSignaturesForAddress(
				context.TODO(),
				solana.MustPublicKeyFromBase58(wallet.Address),
			)
			if err != nil {
				panic(err)
			}
			for _, sig := range out {
				tAmount, newTrans := sr.GetTransactionAmount(sig.Signature.String(), wallet.Address)
				if newTrans {
					sr.AddSolanaTransaction(wallet.Address, sig.Signature.String(), tAmount)
				} else {
					fmt.Println("SOL: No new", wallet.Address[:7]+"... txs.")
				}

				time.Sleep(6 * time.Second)
			}
			time.Sleep(5 * time.Second)
		}
	}

}

func (sr *SolRepository) GetTransactionsForAddressesFirst() {
	for _, wallet := range solWallets {
		endpoint := rpc.MainNetBeta_RPC
		client := rpc.New(endpoint)
		out, err := client.GetSignaturesForAddress(
			context.TODO(),
			solana.MustPublicKeyFromBase58(wallet.Address),
		)
		if err != nil {
			panic(err)
		}

		for _, sig := range out {
			sr.AddSolanaTransactionStart(wallet.Address, sig.Signature.String())
		}

		time.Sleep(5 * time.Second)
	}

}

func (sr *SolRepository) AddSolanaTransactionStart(addr, sig string) {
	// Create a new transaction object
	transaction := Transaction{
		Address:   addr,
		Signature: sig,
	}
	transactions = append(transactions, transaction)
}

func (sr *SolRepository) AddSolanaTransaction(addr, sig string, amount int64) {
	// Create a new transaction object
	transaction := Transaction{
		Address:   addr,
		Signature: sig,
		Amount:    amount,
	}
	if amount <= 50000 { //prevent spam and txs out from slowing down search
		return
	}

	fmt.Println("SOL: "+addr[:5]+"... Recieved:", amount, "lamport.")
	transactions = append(transactions, transaction)
}

func (sr *SolRepository) CreatePendingSolDono(name string, message string, mediaURL string, amountNeeded float64) SuperChat {
	pendingDono := SuperChat{
		Name:         name,
		Message:      message,
		MediaURL:     mediaURL,
		AmountNeeded: amountNeeded,
		Completed:    false,
		CreatedAt:    time.Now().String(),
		CheckedAt:    time.Now().String(),
		CryptoCode:   "SOL",
	}
	return pendingDono
}

func (sr *SolRepository) ContainsTransaction(sig string) bool {
	// searches in reverse order in order to search newest transactions first to avoid needless loops
	for i := len(transactions) - 1; i >= 0; i-- {
		if transactions[i].Signature == sig {
			return true
		}
	}
	return false
}

func (sr *SolRepository) CheckSameBalanceSol(wallet SolWallet) (SolWallet, bool) {
	amt, _ := sr.GetSOLBalance(wallet.Address)
	if wallet.Amount <= 1 {
		wallet.Amount = amt
		return wallet, true
	}
	if amt == wallet.Amount {
		return wallet, true
	} else {
		wallet.Amount = amt
		return wallet, false
	}

}

func (sr *SolRepository) GetSOLBalance(address string) (float64, error) {
	balance, err := solClient.GetBalance(
		context.TODO(), // request context
		address,        // wallet to fetch balance for
	)
	if err != nil {
		return 0, err
	}
	return float64(balance) / 1e9, nil
}

func (sr *SolRepository) GetTransactionAmount(sig, addr string) (int64, bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			fmt.Println("Sleeping 10 seconds.")
			time.Sleep(10 * time.Second)
		}
	}()

	if !sr.ContainsTransaction(sig) {
		url := "https://api.mainnet-beta.solana.com"
		requestBody := fmt.Sprintf(`
  {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "getTransaction",
    "params": [
      "%s",
      "json"
    ]
  }`, sig)

		// Create an HTTP POST request with the request body
		req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(requestBody)))
		if err != nil {
			fmt.Println("Error creating HTTP request:", err)
			return 0, false
		}

		// Set the request header
		req.Header.Set("Content-Type", "application/json")

		// Send the HTTP request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending HTTP request:", err)
			return 0, false
		}
		defer resp.Body.Close()

		// Read the response body
		var responseBody bytes.Buffer
		_, err = responseBody.ReadFrom(resp.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return 0, false
		}

		// Parse the response into a TransactionResponse struct
		var tr TransactionResponse
		err = json.Unmarshal(responseBody.Bytes(), &tr)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return 0, false
		}

		initialAmount := tr.Result.Meta.PreBalances[0]
		endingAmount := tr.Result.Meta.PostBalances[0]
		fromAddr := tr.Result.Transaction.Message.AccountKeys[0]
		fee := tr.Result.Meta.Fee
		endingPlusFee := endingAmount + fee
		amountSent := initialAmount - endingPlusFee
		if fromAddr == addr {
			amountSent *= -1
		}

		//printSolTx(fromAddr, addr, tr.Result.Transaction.Message.AccountKeys[1], amountSent, sig)
		return amountSent, true
	}
	return 0, false
}

func (sr *SolRepository) PrintSolTx(fromAddr, checkAddr, toAddr string, amountSent int64, sig string) {

	decAmountSent := decimal.NewFromInt(amountSent)
	decMultiplier := decimal.NewFromFloat(0.000000001)
	amt := decAmountSent.Mul(decMultiplier)

	if fromAddr == checkAddr {
		fmt.Println("\nTRANSACTION OUT:")
	} else {
		fmt.Println("\nTRANSACTION IN:")
	}
	fmt.Println("To:", toAddr[:7])
	fmt.Println("Sent:", amt)
	fmt.Println("sig:", sig[:7])
}
