package models

import (
	"database/sql"
	"fmt"
	"math"
	"shadowchat/utils"
	"strconv"

	//	"github.com/davecgh/go-spew/spew"

	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var ServerMinMediaDono = 5
var ServerMediaEnabled = true
var killDono = 20.00 * time.Minute // hours it takes for a dono to be unfulfilled before it is no longer checked.
var baseCheckingRate = 25
var prices = CryptoPrice{}

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

type OBSDataStruct struct {
	Username    string
	FilenameGIF string
	FilenameMP3 string
	URLdisplay  string
	URLdonobar  string
	Message     string
	Needed      float64
	Sent        float64
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

type DonoRepository struct {
	db           *sql.DB
	transferRepo *TransferRepository
	userRepo     *UserRepository
	moneroRepo   *MoneroRepository
	pb           ProgressbarData
	donos        map[int]Dono
}

func NewDonoRepository(db *sql.DB, tr *TransferRepository, ur *UserRepository, mr *MoneroRepository) *DonoRepository {
	return &DonoRepository{
		db:           db,
		transferRepo: tr,
		userRepo:     ur,
		moneroRepo:   mr,
		pb:           ProgressbarData{},
		donos:        make(map[int]Dono),
	}
}

// old: checkDonos
func (dr *DonoRepository) check() error {
	for {
		log.Println("Checking donos via checkDonos()")
		fulfilledDonos := dr.checkUnfulfilled()
		if len(fulfilledDonos) > 0 {
			fmt.Println("Fulfilled Donos:")
		}

		for _, dono := range fulfilledDonos {
			fmt.Println(dono)
			user, _ := dr.userRepo.getByID(dono.UserID)
			if user.BillingData.AmountTotal >= 500 {
				user.BillingData.AmountThisMonth += dono.USDAmount
			} else if user.BillingData.AmountTotal+dono.USDAmount >= 500 {
				user.BillingData.AmountThisMonth += user.BillingData.AmountTotal + dono.USDAmount - 500
			}
			user.BillingData.AmountTotal += dono.USDAmount
			dr.userRepo.update(user)

			err := dr.createNewQueueEntry(db, dono.UserID, dono.Address, dono.Name, dono.Message, dono.AmountSent, dono.CurrencyType, dono.USDAmount, dono.MediaURL)
			if err != nil {
				panic(err)
			}

		}
		time.Sleep(time.Duration(25) * time.Second)
	}
}

// old: createNewDono
func (dr *DonoRepository) create(dono *Dono) int64 {
	// Open a new database connection
	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Get current time
	createdAt := time.Now().UTC()

	valid, media_url_ := utils.CheckDonoForMediaUSDThreshold(dono.MediaURL, dono.USDAmount, 5)

	if valid == false {
		media_url_ = ""
	}

	amount_to_send, _ := utils.StandardizeString(dono.AmountToSend)

	// Execute the SQL INSERT statement
	result, err := db.Exec(`
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

func (dr *DonoRepository) updateInMap(updatedDono Dono) {
	if _, ok := dr.donos[updatedDono.ID]; ok {
		// The dono exists in the map, update it
		dr.donos[updatedDono.ID] = updatedDono
	} else {
		// The dono does not exist in the map, handle the error or do nothing
	}
}

func (dr *DonoRepository) updateDonosInDB() {
	// Open a new database connection
	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Loop through the donosMap and update the database with any changes
	for _, dono := range donosMap {
		if dono.Fulfilled && dono.AmountSent != "0.0" {
			log.Println("DONO COMPLETED! Dono: ", dono.AmountSent, dono.CurrencyType)
		}
		_, err = db.Exec("UPDATE donos SET user_id=?, dono_address=?, dono_name=?, dono_message=?, amount_to_send=?, amount_sent=?, currency_type=?, anon_dono=?, fulfilled=?, encrypted_ip=?, created_at=?, updated_at=?, usd_amount=?, media_url=? WHERE dono_id=?", dono.UserID, dono.Address, dono.Name, dono.Message, dono.AmountToSend, dono.AmountSent, dono.CurrencyType, dono.AnonDono, dono.Fulfilled, dono.EncryptedIP, dono.CreatedAt, dono.UpdatedAt, dono.USDAmount, dono.MediaURL, dono.ID)
		if err != nil {
			log.Printf("Error updating Dono with ID %d in the database: %v\n", dono.ID, err)
		}
	}
}

func (dr *DonoRepository) isDonoFulfilled(donoID int) bool {
	// Open a new database connection
	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Retrieve the dono with the given ID
	row := db.QueryRow("SELECT fulfilled FROM donos WHERE dono_id = ?", donoID)

	var fulfilled bool
	err = row.Scan(&fulfilled)
	if err != nil {
		panic(err)
	}

	return fulfilled
}

// old: checkUnfulfilledDonos
func (dr *DonoRepository) checkUnfulfilled() []Dono {
	ips, _ := dr.getUnfulfilledDonoIPs() // get ips

	dr.updatePendingDonos()

	var fulfilledDonos []Dono

	eth_addresses := dr.transferRepo.getAddresses()
	for _, eth_address := range eth_addresses {
		log.Println("Getting ETH txs for:", eth_address)
	}

	tempMap := make(map[string]bool)
	for _, eth_address := range eth_addresses {
		log.Println("Getting ETH txs for:", eth_address)
		transactions, newTX, _ := dr.transferRepo.getEth(eth_address)
		if newTX {
			for _, tx := range transactions {
				if _, exists := tempMap[tx.Hash]; !exists {
					eth_transactions := append(dr.transferRepo.transfers, tx)
					tempMap[tx.Hash] = true
					dr.transferRepo.transfers = eth_transactions
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
				dr.updateInMap(dono)
				continue
			}
		}

		if dono.CurrencyType != "XMR" && dono.CurrencyType != "SOL" {
			// Check if amount matches a completed dono amount
			for _, transaction := range dr.transferRepo.transfers {
				tN := dr.transferRepo.GetTransactionToken(transaction)
				if tN == dono.CurrencyType {
					valueStr := fmt.Sprintf("%.18f", transaction.Value)
					valueToCheck, _ := utils.StandardizeString(dono.AmountToSend)
					log.Println("TX checked:", tN)
					log.Println("Needed:", valueToCheck)
					log.Println("Got   :", valueStr)
					if valueStr == valueToCheck {
						log.Println("Matching TX!")
						dono.AmountSent = valueStr
						dr.addDonoToDonoBar(dono.AmountSent, dono.CurrencyType, dono.UserID) // change Amount To Send to USD value of sent
						dono.Fulfilled = true
						dono.UpdatedAt = time.Now().UTC()
						fulfilledDonos = append(fulfilledDonos, dono)
						dr.updateInMap(dono)
						break
					}
				}
			}

			valueToCheck, _ := utils.ConvertStringTo18DecimalPlaces(dono.AmountToSend)
			dono.UpdatedAt = time.Now().UTC()
			fmt.Println(valueToCheck, dono.CurrencyType, "Dono incomplete.")
			dr.updateInMap(dono)
			continue
		}

		// Check if the dono needs to be skipped based on exponential backoff
		secondsElapsedSinceLastCheck := time.Since(dono.UpdatedAt).Seconds()
		dono.UpdatedAt = time.Now().UTC()

		expoAdder := utils.ReturnIPPenalty(ips, dono.EncryptedIP) + time.Since(dono.CreatedAt).Seconds()/60/60/19
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
			dr.printInfo(dono, secondsElapsedSinceLastCheck, secondsNeededToCheck)
			if dono.AmountSent == xmrNeededStr {
				dono.AmountSent, _ = utils.PruneStringByDecimalPoints(dono.AmountToSend, 5)
				dr.addDonoToDonoBar(dono.AmountSent, dono.CurrencyType, dono.UserID)
				dono.Fulfilled = true
				fulfilledDonos = append(fulfilledDonos, dono)
				dr.updateInMap(dono)
				continue
			}
		} else if dono.CurrencyType == "SOL" {
			if utils.CheckTransactionSolana(dono.AmountToSend, dono.Address, 100) {
				dono.AmountSent, _ = utils.PruneStringByDecimalPoints(dono.AmountToSend, 5)
				dr.addDonoToDonoBar(dono.AmountSent, dono.CurrencyType, dono.UserID) // change Amount To Send to USD value of sent
				dono.Fulfilled = true
				fulfilledDonos = append(fulfilledDonos, dono)
				dr.updateInMap(dono)
				continue
			}
		}
	}
	dr.updateDonosInDB()
	dr.removeFulfilled()
	return fulfilledDonos
}

func (dr *DonoRepository) getUnfulfilledDonoIPs() ([]string, error) {
	ips := []string{}

	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT ip FROM donos WHERE fulfilled = false`)
	if err != nil {
		return ips, err
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
func (dr *DonoRepository) removeFulfilled() {
	for _, dono := range dr.donos {
		if _, ok := donosMap[dono.ID]; ok {
			delete(donosMap, dono.ID)
		}
	}
}

func (dr *DonoRepository) updatePendingDonos() {
	// Open a new database connection
	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Retrieve all unfulfilled donos from the database
	rows, err := db.Query(`SELECT * FROM donos WHERE fulfilled = false`)
	if err != nil {
		panic(err)
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
			panic(err)
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

		dr.addToDonosMap(dono)
	}
}

func (dr *DonoRepository) addToDonosMap(dono Dono) {
	_, ok := dr.donos[dono.ID]
	if !ok {
		dr.donos[dono.ID] = dono
	}
}

func (dr *DonoRepository) createNewQueueEntry(db *sql.DB, user_id int, address string, name string, message string, amount string, currency string, dono_usd float64, media_url string) error {
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		panic(err)
	}
	// Round the amount to 6 decimal places if it has more than 6 decimal places
	if math.Abs(f-math.Round(f)) >= 0.000001 {
		f = math.Round(f*1e6) / 1e6
	}

	embedLink := utils.FormatMediaURL(media_url)

	_, err = db.Exec(`
		INSERT INTO queue (name, message, amount, currency, usd_amount, media_url, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, name, message, amount, currency, dono_usd, embedLink, user_id)
	if err != nil {
		return err
	}
	return nil
}

func (dr *DonoRepository) printInfo(dono Dono, secondsElapsed, secondsNeededToCheck float64) {
	log.Println("Dono ID:", dono.ID, "Address:", dono.Address, "Name:", dono.Name, "To User:", dono.UserID)
	log.Println("Message:", dono.Message)
	fmt.Println(dono.CurrencyType, "Needed:", dono.AmountToSend, "Recieved:", dono.AmountSent)

	log.Println("Time since check:", fmt.Sprintf("%.2f", secondsElapsed), "Needed:", fmt.Sprintf("%.2f", secondsNeededToCheck))

}
func (dr *DonoRepository) addDonoToDonoBar(as, c string, userID int) {
	f, err := strconv.ParseFloat(as, 64)
	usdVal := getUSDValue(f, c)
	obsData, err := getOBSDataByUserID(userID)
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

	err = updateObsData(db, userID, obsData.FilenameGIF, obsData.FilenameMP3, "alice", dr.pb)

	if err != nil {
		log.Println("Error: ", err)
	}
}

func (dr *DonoRepository) replayDono(donation Donation, userID int) {
	valid, media_url_ := utils.CheckDonoForMediaUSDThreshold(donation.DonationMedia, utils.ConvertToFloat64(donation.USDValue), 5)

	if valid == false {
		media_url_ = ""
	}

	err := dr.createNewQueueEntry(db, userID, "ReplayAddress", donation.DonationName, donation.DonationMessage, donation.AmountSent, donation.Crypto, utils.ConvertToFloat64(donation.USDValue), media_url_)
	if err != nil {
		panic(err)
	}
}

func updateObsData(db *sql.DB, userID int, gifName string, mp3Name string, ttsVoice string, pbData ProgressbarData) error {

	updateObsData := `
        UPDATE obs
        SET user_id = ?,
            gif_name = ?,
            mp3_name = ?,
            tts_voice = ?,
            message = ?,
            needed = ?,
            sent = ?
        WHERE id = ?;`
	_, err := db.Exec(updateObsData, userID, gifName, mp3Name, ttsVoice, pbData.Message, pbData.Needed, pbData.Sent, userID)
	return err
}

func getUSDValue(as float64, c string) float64 {
	usdVal := 0.00

	priceMap := map[string]float64{
		"XMR":   prices.Monero,
		"SOL":   prices.Solana,
		"ETH":   prices.Ethereum,
		"PAINT": prices.Paint,
		"HEX":   prices.Hexcoin,
		"MATIC": prices.Polygon,
		"BUSD":  prices.BinanceUSD,
		"SHIB":  prices.ShibaInu,
		"PNK":   prices.Kleros,
	}

	if price, ok := priceMap[c]; ok {
		usdVal = as * price
	} else {
		usdVal = 1.00
		return usdVal
	}
	usdValStr := fmt.Sprintf("%.2f", usdVal)      // format usdVal as a string with 2 decimal points
	usdVal, _ = strconv.ParseFloat(usdValStr, 64) // convert the string back to a float

	return usdVal
}

func getOBSDataByUserID(userID int) (utils.OBSDataStruct, error) {
	var obsData utils.OBSDataStruct
	//var alertURL sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := db.QueryRow("SELECT gif_name, mp3_name, `message`, needed, sent FROM obs WHERE user_id=?", userID)

	err := row.Scan(&obsData.FilenameGIF, &obsData.FilenameMP3, &obsData.Message, &obsData.Needed, &obsData.Sent)
	if err != nil {
		log.Println("Couldn't get obsData,", err)
		return obsData, err
	}

	return obsData, nil

}
