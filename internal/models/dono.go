package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"shadowchat/utils"
	"strconv"
	"strings"

	//	"github.com/davecgh/go-spew/spew"

	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shopspring/decimal"
)

var ServerMinMediaDono = 5
var ServerMediaEnabled = true
var killDono = 20.00 * time.Minute // hours it takes for a dono to be unfulfilled before it is no longer checked.
var baseCheckingRate = 25
var prices = CryptoPrice{}

type TempERCTransaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	From              string `json:"from"`
	ContractAddress   string `json:"contractAddress"`
	To                string `json:"to"`
	Value             string `json:"value"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Input             string `json:"input"`
	Confirmations     string `json:"confirmations"`
}

type Response struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  struct {
		Transfers []Transfer `json:"transfers"`
	} `json:"result"`
}

type ERCResponse struct {
	Status  string               `json:"status"`
	Message string               `json:"message"`
	Result  []TempERCTransaction `json:"result"`
}

type TempETHTransaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	TransactionIndex  string `json:"transactionIndex"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	IsError           string `json:"isError"`
	Txreceipt_status  string `json:"txreceipt_status"`
	Input             string `json:"input"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	GasUsed           string `json:"gasUsed"`
	Confirmations     string `json:"confirmations"`
	MethodId          string `json:"methodId"`
	FunctionName      string `json:"functionName"`
}

type EthResponse struct {
	Status  string               `json:"status"`
	Message string               `json:"message"`
	Result  []TempETHTransaction `json:"result"`
}

type RawContract struct {
	Value   string `json:"value"`
	Address string `json:"address"`
	Decimal string `json:"decimal"`
}

type Transfer struct {
	BlockNum        string      `json:"blockNum"`
	UniqueId        string      `json:"uniqueId"`
	Hash            string      `json:"hash"`
	From            string      `json:"from"`
	To              string      `json:"to"`
	Value           float64     `json:"value"`
	Erc721TokenId   interface{} `json:"erc721TokenId"`
	Erc1155Metadata interface{} `json:"erc1155Metadata"`
	TokenId         interface{} `json:"tokenId"`
	Asset           string      `json:"asset"`
	Category        string      `json:"category"`
	RawContract     RawContract `json:"rawContract"`
}
type CryptoPrice struct {
	Monero     float64 `json:"monero"`
	Solana     float64 `json:"solana"`
	Ethereum   float64 `json:"ethereum"`
	Paint      float64 `json:"paint"`
	Hexcoin    float64 `json:"hex"`
	Polygon    float64 `json:"matic"`
	BinanceUSD float64 `json:"binance-usd"`
	ShibaInu   float64 `json:"shiba-inu"`
	Kleros     float64 `json:"pnk"`
	WBTC       float64 `json:"wbtc"`
	TUSD       float64 `json: "tusd"`
}

var contracts = map[string]string{
	"PAINT":     "0x4c6ec08cf3fc987c6c4beb03184d335a2dfc4042",
	"HEX":       "0x2b591e99afE9f32eAA6214f7B7629768c40Eeb39",
	"MATIC":     "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0",
	"BUSD":      "0x4Fabb145d64652a948d72533023f6E7A623C7C53",
	"SHIBA_INU": "0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE",
	"PNK":       "0x93ed3fbe21207ec2e8f2d3c3de6e058cb73bc04d",
}

type ViewDonosData struct {
	Username string
	Donos    []Dono
}

type ProgressbarData struct {
	Message string
	Needed  float64
	Sent    float64
	Refresh int
}

type AlertPageData struct {
	Name          string
	Message       string
	Amount        float64
	Currency      string
	MediaURL      string
	USDAmount     float64
	Refresh       int
	DisplayToggle string
	Userpath      string
}

type Donation struct {
	ID              string `json:"donoID"`
	DonationName    string `json:"donationName"`
	DonationMessage string `json:"donationMessage"`
	DonationMedia   string `json:"donationMedia"`
	USDValue        string `json:"usdValue"`
	AmountSent      string `json:"amountSent"`
	Crypto          string `json:"crypto"`
}

type Dono struct {
	ID           int
	UserID       int
	Address      string
	Name         string
	Message      string
	AmountToSend string
	AmountSent   string
	CurrencyType string
	AnonDono     bool
	Fulfilled    bool
	EncryptedIP  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	USDAmount    float64
	MediaURL     string
}

type DonoRepositoryInterface interface {
	Check() error
	Create(dono *Dono) int64
	UpdateInMap(updatedDono Dono)
	UpdateDonosInDB()
	IsDonoFulfilled(donoID int) bool
	CheckUnfulfilled() ([]Dono, error)
	GetUnfulfilledDonoIPs() ([]string, error)
	RemoveFulfilled()
	UpdatePendingDonos() error
	AddToDonosMap(dono Dono)
	CreateNewQueueEntry(db *sql.DB, user_id int, address string, name string, message string, amount string, currency string, dono_usd float64, media_url string) error
	PrintInfo(dono Dono, secondsElapsed, secondsNeededToCheck float64)
	AddDonoToDonoBar(as, c string, userID int)
	ReplayDono(donation Donation, userID int)
	GetEthAddresses() []string
	CheckObsData() (bool, error)
	GetEth(eth_address string) ([]Transfer, bool, error)
	CheckNewETHTransactions(eth_address string) bool
	CheckNewERCTransactions(eth_address string) bool
	GetEthTransactions(eth_address string) ([]Transfer, error)
	GetTransactionAmount(t Transfer) string
	GetTransactionToken(t Transfer) string
	GetTokenName(contractAddr string) string
}

type DonoRepository struct {
	db                *sql.DB
	userRepo          *UserRepository
	moneroRepo        *MoneroRepository
	pb                ProgressbarData
	donos             map[int]Dono
	transfers         []Transfer
	transactionMap    map[string]Transfer
	ethAddresses      map[string][]Transfer
	addressMap        map[string]bool
	addresses         []string
	ethTransactions   map[string][]TempETHTransaction
	erc20Transactions map[string][]TempERCTransaction
}

func NewDonoRepository(db *sql.DB, ur *UserRepository, mr *MoneroRepository) *DonoRepository {
	return &DonoRepository{
		db:                db,
		userRepo:          ur,
		moneroRepo:        mr,
		pb:                ProgressbarData{},
		donos:             make(map[int]Dono),
		transfers:         []Transfer{},
		transactionMap:    make(map[string]Transfer),
		ethAddresses:      map[string][]Transfer{},
		addressMap:        make(map[string]bool),
		addresses:         []string{},
		ethTransactions:   make(map[string][]TempETHTransaction),
		erc20Transactions: make(map[string][]TempERCTransaction),
	}
}

// old: checkDonos
func (dr *DonoRepository) Check() error {
	for {
		log.Println("Checking pending wallets for successful starting")
		for _, u_ := range dr.userRepo.users {
			if u_.WalletUploaded && u_.WalletPending {
				u_.WalletPending = dr.moneroRepo.CheckMoneroPort(u_.UserID)
				if !u_.WalletPending {
					dr.userRepo.update(u_)
				}
			}
		}

		log.Println("Checking donos via checkDonos()")
		fulfilledDonos, _ := dr.CheckUnfulfilled()
		if len(fulfilledDonos) > 0 {
			fmt.Println("Fulfilled Donos:")
		}

		for _, dono := range fulfilledDonos {
			fmt.Println(dono)
			user := dr.userRepo.users[dono.UserID]
			if user.BillingData.AmountTotal >= 500 {
				user.BillingData.AmountThisMonth += dono.USDAmount
			} else if user.BillingData.AmountTotal+dono.USDAmount >= 500 {
				user.BillingData.AmountThisMonth += user.BillingData.AmountTotal + dono.USDAmount - 500
			}
			user.BillingData.AmountTotal += dono.USDAmount
			dr.userRepo.update(user)

			err := dr.CreateNewQueueEntry(dono.UserID, dono.Address, dono.Name, dono.Message, dono.AmountSent, dono.CurrencyType, dono.USDAmount, dono.MediaURL)
			if err != nil {
				panic(err)
			}

		}
		time.Sleep(time.Duration(25) * time.Second)
	}
}

// old: createNewDono
func (dr *DonoRepository) Create(dono *Dono) int64 {
	// Get current time
	createdAt := time.Now().UTC()

	valid, media_url_ := CheckDonoForMediaUSDThreshold(dono.MediaURL, dono.USDAmount)

	if valid == false {
		media_url_ = ""
	}

	amount_to_send, _ := utils.StandardizeString(dono.AmountToSend)

	// Execute the SQL INSERT statement
	result, err := dr.db.Exec(`
        INSERT INTO donos (
            user_id,
            dono_address,
            dono_name,
            dono_message,
            amount_to_send,
            amount_sent,
            currency_type,
            anon_dono,
            fulfilled,
            encrypted_ip,
            created_at,
            updated_at,
            usd_amount,
            media_url
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `,
		dono.UserID,
		dono.Address,
		dono.Name,
		dono.Message,
		amount_to_send,
		"0.0",
		dono.CurrencyType,
		dono.AnonDono,
		false,
		dono.EncryptedIP,
		createdAt,
		createdAt,
		dono.USDAmount,
		media_url_,
	)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	// Get the id of the newly created dono
	id, err := result.LastInsertId()
	if err != nil {
		log.Println(err)
		panic(err)
	}

	return id
}

func (dr *DonoRepository) UpdateInMap(updatedDono Dono) {
	if _, ok := dr.donos[updatedDono.ID]; ok {
		// The dono exists in the map, update it
		dr.donos[updatedDono.ID] = updatedDono
	} else {
		// The dono does not exist in the map, log an error
		log.Printf("Failed to update dono with ID %d. Dono does not exist in the map.", updatedDono.ID)
	}
}

func (dr *DonoRepository) UpdateDonosInDB() {
	// Loop through the donosMap and update the database with any changes
	for _, dono := range dr.donos {
		if dono.Fulfilled && dono.AmountSent != "0.0" {
			log.Println("DONO COMPLETED! Dono: ", dono.AmountSent, dono.CurrencyType)
		}
		_, err := dr.db.Exec("UPDATE donos SET user_id=?, dono_address=?, dono_name=?, dono_message=?, amount_to_send=?, amount_sent=?, currency_type=?, anon_dono=?, fulfilled=?, encrypted_ip=?, created_at=?, updated_at=?, usd_amount=?, media_url=? WHERE dono_id=?", dono.UserID, dono.Address, dono.Name, dono.Message, dono.AmountToSend, dono.AmountSent, dono.CurrencyType, dono.AnonDono, dono.Fulfilled, dono.EncryptedIP, dono.CreatedAt, dono.UpdatedAt, dono.USDAmount, dono.MediaURL, dono.ID)
		if err != nil {
			log.Printf("Error updating Dono with ID %d in the database: %v\n", dono.ID, err)
		}
	}
}

func (dr *DonoRepository) IsDonoFulfilled(donoID int) bool {
	// Retrieve the dono with the given ID
	row := dr.db.QueryRow("SELECT fulfilled FROM donos WHERE dono_id = ?", donoID)

	var fulfilled bool
	err := row.Scan(&fulfilled)
	if err != nil {
		panic(err)
	}

	return fulfilled
}

// old: checkUnfulfilledDonos
func (dr *DonoRepository) CheckUnfulfilled() ([]Dono, error) {
	ips, err := dr.GetUnfulfilledDonoIPs() // get ips

	if err != nil {
		return nil, fmt.Errorf("failed to get unfulfilled dono IPs: %w", err)
	}

	dr.UpdatePendingDonos()

	var fulfilledDonos []Dono

	eth_addresses := dr.GetEthAddresses()
	for _, eth_address := range eth_addresses {
		log.Println("Getting ETH txs for:", eth_address)
	}

	tempMap := make(map[string]bool)
	for _, eth_address := range eth_addresses {
		log.Println("Getting ETH txs for:", eth_address)
		transactions, newTX, _ := dr.GetEth(eth_address)
		if newTX {
			for _, tx := range transactions {
				log.Println("new tx", tx)
				if _, exists := tempMap[tx.Hash]; !exists {
					eth_transactions := append(dr.transfers, tx)
					tempMap[tx.Hash] = true
					dr.transfers = eth_transactions
				}
			}
			time.Sleep(2 * time.Second)
		}

	}

	for _, dono := range dr.donos {
		// Check if the dono has exceeded the killDono time
		if !dono.Fulfilled {
			timeElapsedFromDonoCreation := time.Since(dono.CreatedAt)
			if timeElapsedFromDonoCreation > killDono || dono.Address == " " || dono.AmountToSend == "0.0" {
				dono.Fulfilled = true
				if dono.Address == " " {
					log.Println("No dono address, killed (marked as fulfilled) and won't be checked again.")
				} else {
					log.Println("Dono too old, killed (marked as fulfilled) and won't be checked again.")
				}
				dr.UpdateInMap(dono)
				continue
			}
		}

		if dono.CurrencyType != "XMR" && dono.CurrencyType != "SOL" {
			// Check if amount matches a completed dono amount
			for _, transaction := range dr.transfers {
				tN := dr.GetTransactionToken(transaction)
				if tN == dono.CurrencyType {
					valueStr := fmt.Sprintf("%.18f", transaction.Value)
					valueToCheck, _ := utils.StandardizeString(dono.AmountToSend)
					log.Println("TX checked:", tN)
					log.Println("Needed:", valueToCheck)
					log.Println("Got   :", valueStr)
					if valueStr == valueToCheck {
						log.Println("Matching TX!")
						dono.AmountSent = valueStr
						dr.AddDonoToDonoBar(dono.AmountSent, dono.CurrencyType, dono.UserID) // change Amount To Send to USD value of sent
						dono.Fulfilled = true
						dono.UpdatedAt = time.Now().UTC()
						fulfilledDonos = append(fulfilledDonos, dono)
						dr.UpdateInMap(dono)
						break
					}
				}
			}

			valueToCheck, _ := utils.ConvertStringTo18DecimalPlaces(dono.AmountToSend)
			dono.UpdatedAt = time.Now().UTC()
			fmt.Println(valueToCheck, dono.CurrencyType, "Dono incomplete.")
			dr.UpdateInMap(dono)
			continue
		}

		// Check if the dono needs to be skipped based on exponential backoff
		secondsElapsedSinceLastCheck := time.Since(dono.UpdatedAt).Seconds()
		dono.UpdatedAt = time.Now().UTC()

		expoAdder := ReturnIPPenalty(ips, dono.EncryptedIP) + time.Since(dono.CreatedAt).Seconds()/60/60/19
		secondsNeededToCheck := math.Pow(float64(baseCheckingRate)-0.02, expoAdder)

		if secondsElapsedSinceLastCheck < secondsNeededToCheck {
			log.Println("Not enough time has passed, skipping.")
			continue // If not enough time has passed then ignore
		}

		log.Println("Enough time has passed, checking.")

		if dono.CurrencyType == "XMR" {
			xmrFl, _ := dr.moneroRepo.getBalance(dono.Address, dono.UserID)
			xmrSent, _ := utils.StandardizeFloatToString(xmrFl)
			dono.AmountSent = xmrSent
			xmrNeededStr, _ := utils.StandardizeString(dono.AmountToSend)
			dr.PrintInfo(dono, secondsElapsedSinceLastCheck, secondsNeededToCheck)
			if dono.AmountSent == xmrNeededStr {
				dono.AmountSent, _ = utils.PruneStringByDecimalPoints(dono.AmountToSend, 5)
				dr.AddDonoToDonoBar(dono.AmountSent, dono.CurrencyType, dono.UserID)
				dono.Fulfilled = true
				fulfilledDonos = append(fulfilledDonos, dono)
				dr.UpdateInMap(dono)
				continue
			}
		} else if dono.CurrencyType == "SOL" {
			if utils.CheckTransactionSolana(dono.AmountToSend, dono.Address, 100) {
				dono.AmountSent, _ = utils.PruneStringByDecimalPoints(dono.AmountToSend, 5)
				dr.AddDonoToDonoBar(dono.AmountSent, dono.CurrencyType, dono.UserID) // change Amount To Send to USD value of sent
				dono.Fulfilled = true
				fulfilledDonos = append(fulfilledDonos, dono)
				dr.UpdateInMap(dono)
				continue
			}
		}
	}
	dr.UpdateDonosInDB()
	dr.RemoveFulfilled()
	return fulfilledDonos, nil
}

func (dr *DonoRepository) GetUnfulfilledDonoIPs() ([]string, error) {
	ips := []string{}

	rows, err := dr.db.Query(`SELECT encrypted_ip FROM donos WHERE fulfilled = false`)
	if err != nil {
		return ips, fmt.Errorf("failed to get unfulfilled dono IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip string
		err := rows.Scan(&ip)
		if err != nil {
			return ips, err
		}
		ips = append(ips, ip)
	}

	err = rows.Err()
	if err != nil {
		return ips, err
	}

	return ips, nil
}
func (dr *DonoRepository) RemoveFulfilled() {
	for _, dono := range dr.donos {
		if _, ok := dr.donos[dono.ID]; ok {
			delete(dr.donos, dono.ID)
		}
	}
}

func (dr *DonoRepository) UpdatePendingDonos() error {

	// Retrieve all unfulfilled donos from the database
	rows, err := dr.db.Query(`SELECT * FROM donos WHERE fulfilled = false`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var dono Dono
		var tmpAmountToSend sql.NullString
		var tmpAmountSent sql.NullString
		var tmpUSDAmount sql.NullFloat64
		var tmpMediaURL sql.NullString

		err := rows.Scan(&dono.ID, &dono.UserID, &dono.Address, &dono.Name, &dono.Message, &tmpAmountToSend, &tmpAmountSent, &dono.CurrencyType, &dono.AnonDono, &dono.Fulfilled, &dono.EncryptedIP, &dono.CreatedAt, &dono.UpdatedAt, &tmpUSDAmount, &tmpMediaURL)
		if err != nil {
			return err
		}

		if tmpUSDAmount.Valid {
			dono.USDAmount = tmpUSDAmount.Float64
		} else {
			dono.USDAmount = 0.0
		}

		if tmpAmountToSend.Valid {
			dono.AmountToSend = tmpAmountToSend.String
		} else {
			dono.AmountToSend = "0.0"
		}

		if tmpAmountSent.Valid {
			dono.AmountSent = tmpAmountSent.String
		} else {
			dono.AmountSent = "0.0"
		}

		if tmpMediaURL.Valid {
			dono.MediaURL = tmpMediaURL.String
		} else {
			dono.MediaURL = ""
		}

		dr.AddToDonosMap(dono)
	}
	return nil
}

func (dr *DonoRepository) AddToDonosMap(dono Dono) {
	_, ok := dr.donos[dono.ID]
	if !ok {
		dr.donos[dono.ID] = dono
	}
}

func (dr *DonoRepository) CreateNewQueueEntry(user_id int, address string, name string, message string, amount string, currency string, dono_usd float64, media_url string) error {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		panic(err)
	}
	// Round the amount to 6 decimal places if it has more than 6 decimal places
	if math.Abs(f-math.Round(f)) >= 0.000001 {
		f = math.Round(f*1e6) / 1e6
	}

	embedLink := FormatMediaURL(media_url)

	_, err = dr.db.Exec(`
		INSERT INTO queue (name, message, amount, currency, usd_amount, media_url, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, name, message, amount, currency, dono_usd, embedLink, user_id)
	if err != nil {
		return err
	}
	return nil
}

func (dr *DonoRepository) PrintInfo(dono Dono, secondsElapsed, secondsNeededToCheck float64) {
	log.Println("Dono ID:", dono.ID, "Address:", dono.Address, "Name:", dono.Name, "To User:", dono.UserID)
	log.Println("Message:", dono.Message)
	fmt.Println(dono.CurrencyType, "Needed:", dono.AmountToSend, "Recieved:", dono.AmountSent)

	log.Println("Time since check:", fmt.Sprintf("%.2f", secondsElapsed), "Needed:", fmt.Sprintf("%.2f", secondsNeededToCheck))

}
func (dr *DonoRepository) AddDonoToDonoBar(as, c string, userID int) {
	f, err := strconv.ParseFloat(as, 64)
	usdVal := GetUSDValue(f, c)
	obsData, err := dr.userRepo.getOBSDataByUserID(userID)
	dr.pb.Sent = obsData.Sent
	dr.pb.Needed = obsData.Needed
	dr.pb.Message = obsData.Message
	dr.pb.Sent += usdVal

	sent, err := strconv.ParseFloat(fmt.Sprintf("%.2f", dr.pb.Sent), 64)
	if err != nil {
		// handle the error here
		log.Println("Error converting to cents: ", err)
	}
	dr.pb.Sent = sent

	amountSent = dr.pb.Sent

	err = dr.userRepo.updateObsData(userID, obsData.FilenameGIF, obsData.FilenameMP3, "alice", dr.pb)

	if err != nil {
		log.Println("Error: ", err)
	}
}

func (dr *DonoRepository) ReplayDono(donation Donation, userID int) {
	valid, media_url_ := CheckDonoForMediaUSDThreshold(donation.DonationMedia, ConvertToFloat64(donation.USDValue))

	if valid == false {
		media_url_ = ""
	}

	err := dr.CreateNewQueueEntry(userID, "ReplayAddress", donation.DonationName, donation.DonationMessage, donation.AmountSent, donation.Crypto, ConvertToFloat64(donation.USDValue), media_url_)
	if err != nil {
		panic(err)
	}
}

// old: returnETHAddresses
func (dr *DonoRepository) GetEthAddresses() []string {
	for _, dono := range dr.donos {
		if dono.CurrencyType != "XMR" && dono.CurrencyType != "SOL" {
			// Check if the address has already been added, and if not, add it to the slice and map
			if _, ok := dr.addressMap[dono.Address]; !ok {
				dr.addressMap[dono.Address] = true
				dr.addresses = append(dr.addresses, dono.Address)
			}
		}
	}

	return dr.addresses
}

func (dr *DonoRepository) CheckObsData() (bool, error) {
	var count int
	err := dr.db.QueryRow("SELECT COUNT(*) FROM obs").Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (dr *DonoRepository) GetEth(eth_address string) ([]Transfer, bool, error) {
	/*check if eth_address is in ethAddresses*/
	if _, exists := dr.ethAddresses[eth_address]; !exists {
		transactions, err := dr.GetEthTransactions(eth_address)
		if err != nil {
			log.Println("eth tx error:", err)
		}
		dr.ethAddresses[eth_address] = transactions
		log.Println("eth address doesn't exist, checking.")
		return dr.ethAddresses[eth_address], true, nil
	} else {
		newTX := false
		log.Println("eth address does exist, check if transactions are the same")
		if dr.CheckNewETHTransactions(eth_address) {
			log.Println("NEW ETH TXS")
			transactions, _ := dr.GetEthTransactions(eth_address)
			dr.ethAddresses[eth_address] = transactions
			dr.transfers = append(dr.transfers, transactions...) // Add transactions to dr.transfers
			newTX = true
			// Handle ERC transactions if needed
		} else if dr.CheckNewERCTransactions(eth_address) {
			log.Println("NEW ERC TXS")
			transactions, _ := dr.GetEthTransactions(eth_address)
			dr.ethAddresses[eth_address] = transactions
			dr.transfers = append(dr.transfers, transactions...) // Add transactions to dr.transfers
			newTX = true
		}
		log.Println("There are no new txs")
		return dr.ethAddresses[eth_address], newTX, nil
	}
}

func (dr *DonoRepository) CheckNewETHTransactions(eth_address string) bool {
	etherscanAPI, err := ioutil.ReadFile("./etherscan_api")
	if err != nil {
		log.Println("Error reading Etherscan API Key:", err)
		return false
	}

	url := "https://api.etherscan.io/api?module=account&action=txlist&address=" +
		eth_address + "&startblock=0&endblock=99999999&sort=asc&apikey=" + string(etherscanAPI)

	url = strings.ReplaceAll(url, "\n", "")

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error sending GET request:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return false
	}

	var response EthResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Error unmarshalling response body:", err)
		return false
	}

	if _, exists := dr.ethTransactions[eth_address]; !exists {
		dr.ethTransactions[eth_address] = response.Result
		log.Println("ethTransactions[eth_address] not found")
		return true
	} else {
		if len(dr.ethTransactions[eth_address]) == len(response.Result) {
			log.Println("ethTransactions[eth_address] found but no new txs")
			return false
		} else {
			dr.ethTransactions[eth_address] = response.Result
			log.Println("ethTransactions[eth_address] found and new txs")
			return true
		}
	}
}

func (dr *DonoRepository) CheckNewERCTransactions(eth_address string) bool {
	etherscanAPI, err := ioutil.ReadFile("./etherscan_api")
	if err != nil {
		log.Println("Error reading Etherscan API Key:", err)
		return false
	}

	url := "https://api.etherscan.io/api?module=account&action=tokentx&address=" +
		eth_address + "&startblock=0&endblock=999999999&sort=asc&apikey=" + string(etherscanAPI)

	url = strings.ReplaceAll(url, "\n", "")

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error sending GET request:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return false
	}

	var response ERCResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Error unmarshalling response body:", err)
		return false
	}

	if _, exists := dr.erc20Transactions[eth_address]; !exists {
		dr.erc20Transactions[eth_address] = response.Result
		log.Println("erc20Transactions[eth_address] not found")
		return true
	} else {
		if len(dr.erc20Transactions[eth_address]) == len(response.Result) {
			log.Println("erc20Transactions[eth_address] found but no new txs")
			return false
		} else {
			log.Println("erc20Transactions[eth_address] found and new txs")
			dr.erc20Transactions[eth_address] = response.Result
			return true
		}
	}
}

func (dr *DonoRepository) GetEthTransactions(eth_address string) ([]Transfer, error) {
	// Read Alchemy API KEY from file
	alchemyAPIKEY, err := ioutil.ReadFile("./alchemy_api")
	if err != nil {
		return nil, err
	}

	url := "https://eth-mainnet.g.alchemy.com/v2/" + string(alchemyAPIKEY)
	url = strings.ReplaceAll(url, "\n", "")

	payload := strings.NewReader("{\"id\":1,\"jsonrpc\":\"2.0\",\"method\":\"alchemy_getAssetTransfers\",\"params\":[{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"toAddress\":\"" + eth_address + "\",\"category\":[\"external\", \"erc20\"],\"withMetadata\":false,\"excludeZeroValue\":true,\"maxCount\":\"0x3e8\",\"order\":\"desc\"}]}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	transfers := response.Result.Transfers
	return transfers, nil
}

func (dr *DonoRepository) GetTransactionAmount(t Transfer) string {
	d := decimal.NewFromFloat(t.Value)
	return d.String()
}

func (dr *DonoRepository) GetTransactionToken(t Transfer) string {
	asset := ""
	if t.RawContract.Address == "" {
		asset = "ETH"
	} else {
		asset = dr.GetTokenName(t.RawContract.Address)
	}
	return asset
}

func (dr *DonoRepository) GetTokenName(contractAddr string) string {
	switch contractAddr {
	case contracts["PAINT"]:
		return "PAINT"
	case contracts["HEX"]:
		return "HEX"
	case contracts["MATIC"]:
		return "MATIC"
	case contracts["BUSD"]:
		return "BUSD"
	case contracts["SHIBA_INU"]:
		return "SHIBA_INU"
	case contracts["PNK"]:
		return "PNK"
	default:
		return "UNKNOWN"
	}
}

func (dr *DonoRepository) checkAccountBillings() {
	for {
		for _, user := range dr.userRepo.users {
			if user.BillingData.Enabled {
				// check if the updated amount is from the current or previous month
				now := time.Now()
				updatedMonth := user.BillingData.UpdatedAt.Month()
				currentMonth := now.Month()
				if user.BillingData.AmountTotal >= 500 {

					if updatedMonth == currentMonth {
						continue
					} else if now.Day() >= 4 {
						// the updated amount is from an earlier month, so disable billing
						user.BillingData.Enabled = false
						user.BillingData.NeedToPay = true
						user.BillingData.AmountNeeded = math.Round(user.BillingData.AmountThisMonth*0.03*100) / 100

						xmrNeeded := GetXMRAmountInUSD(user.BillingData.AmountNeeded)
						ethNeeded := GetETHAmountInUSD(user.BillingData.AmountNeeded)

						PayID, PayAddress := dr.moneroRepo.getNewAccountXMR()
						user.BillingData.XMRPayID = PayID
						user.BillingData.XMRAddress = PayAddress
						user.BillingData.XMRAmount = xmrNeeded
						user.BillingData.ETHAmount = ethNeeded

						fmt.Printf("Disabling billing for user %d\n", user.UserID)
					} else {
						user.BillingData.NeedToPay = true
						user.BillingData.Enabled = true
					}
				} else {
					dr.userRepo.renewUserSubscription(user)
				}
			}
		}
		time.Sleep(time.Duration(12) * time.Hour)
	}
}

func (dr *DonoRepository) CreateTestDono(user_id int, name string, curr string, message string, amount string, usdAmount float64, media_url string) {
	valid, media_url_ := dr.CheckDonoForMediaUSDThreshold(media_url, usdAmount)

	if valid == false {
		media_url_ = ""
	}

	log.Println("TESTING DONO IN FIVE SECONDS")
	time.Sleep(5 * time.Second)
	log.Println("TESTING DONO NOW")
	err := dr.CreateNewQueueEntry(user_id, "TestAddress", name, message, amount, curr, usdAmount, media_url_)
	if err != nil {
		panic(err)
	}
	dr.AddDonoToDonoBar(amount, curr, user_id)
}

func (dr *DonoRepository) CheckDonoForMediaUSDThreshold(media_url string, dono_usd float64) (bool, string) {
	valid := true
	if dono_usd < float64(ServerMinMediaDono) {
		media_url = ""
		valid = false

	}
	return valid, media_url
}
