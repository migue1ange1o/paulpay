package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var starting_port int = 28088

var cryptoMap = map[string]map[string]interface{}{
	"paint": {
		"name":     "Paint",
		"code":     "PAINT",
		"svg":      "paint.svg",
		"min":      "{{.MinPaint}}",
		"contract": contracts["PAINT"],
		"decimals": 18,
	},
	"hex": {
		"name":     "Hexcoin",
		"code":     "HEX",
		"svg":      "hex.svg",
		"min":      "{{.MinHex}}",
		"contract": contracts["HEX"],
		"decimals": 8,
	},
	"matic": {
		"name":     "Polygon",
		"code":     "MATIC",
		"svg":      "matic.svg",
		"min":      "{{.MinPolygon}}",
		"contract": contracts["MATIC"],
		"decimals": 18,
	},
	"busd": {
		"name":     "Binance USD",
		"code":     "BUSD",
		"svg":      "busd.svg",
		"min":      "{{.MinBusd}}",
		"contract": contracts["BUSD"],
		"decimals": 18,
	},
	"shiba_inu": {
		"name":     "Shiba Inu",
		"code":     "SHIB",
		"svg":      "shiba_inu.svg",
		"min":      "{{.MinShib}}",
		"contract": contracts["SHIBA_INU"],
		"decimals": 18,
	},
	"pnk": {
		"name":     "Kleros",
		"code":     "PNK",
		"svg":      "pnk.svg",
		"min":      "{{.MinPnk}}",
		"contract": contracts["PNK"],
		"decimals": 18,
	},
}

type IndexDisplay struct {
	MaxChar        int
	MinDono        int
	MinSolana      float64
	MinMonero      float64
	MinEthereum    float64
	MinPaint       float64
	MinHex         float64
	MinPolygon     float64
	MinBusd        float64
	MinShib        float64
	MinPnk         float64
	SolPrice       float64
	XMRPrice       float64
	ETHPrice       float64
	PaintPrice     float64
	HexPrice       float64
	PolygonPrice   float64
	BusdPrice      float64
	ShibPrice      float64
	PnkPrice       float64
	MinAmnt        float64
	WalletPending  bool
	Links          string
	Checked        string
	CryptosEnabled CryptosEnabled
	DefaultCrypto  string
	Username       string
}

func CreateDatabaseIfNotExists(db *sql.DB, ur *UserRepository) error {
	// create the tables if they don't exist
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS donos (
            dono_id INTEGER PRIMARY KEY,
            user_id INTEGER,
            dono_address TEXT,
            dono_name TEXT,
            dono_message TEXT,
            amount_to_send TEXT,            
            amount_sent TEXT,
            currency_type TEXT,
            anon_dono BOOL,
            fulfilled BOOL,
            encrypted_ip TEXT,
            created_at DATETIME,
            updated_at DATETIME,
            FOREIGN KEY(user_id) REFERENCES users(id)
        )
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS billing (
        	billing_id INTEGER PRIMARY KEY,
            user_id INTEGER,
            amount_this_month FLOAT,
            amount_total FLOAT,
            enabled BOOL,
            need_to_pay BOOL,
            eth_amount TEXT,
            xmr_amount TEXT,
            xmr_pay_id TEXT,
            created_at DATETIME,
            updated_at DATETIME,
            FOREIGN KEY(user_id) REFERENCES users(id)
        )
    `)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS addresses (
            key_public TEXT NOT NULL,
            key_private BLOB NOT NULL
        )
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS queue (
            name TEXT,
            message TEXT,
            amount FLOAT,
            currency TEXT
        )
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT UNIQUE,
            HashedPassword BLOB,
            eth_address TEXT,
            sol_address TEXT,
            hex_address TEXT,
            xmr_wallet_password TEXT,
            min_donation_threshold FLOAT,
            min_media_threshold FLOAT,
            media_enabled BOOL,
            created_at DATETIME,
            modified_at DATETIME,
            links TEXT,
            dono_gif TEXT,
            dono_sound TEXT,
            alert_url TEXT,
            date_enabled DATETIME,
            wallet_uploaded BOOL,
            cryptos_enabled TEXT

        )
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS pendingusers (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT UNIQUE,
            HashedPassword BLOB,
            XMRPayID TEXT,
            XMRNeeded TEXT,
            ETHNeeded TEXT
        )
    `)

	if err != nil {
		return err
	}

	err = CreateObsTable(db)
	if err != nil {
		log.Fatal(err)
	}

	ur.CreateAdmin()
	ur.CreateNew("paul", "hunter")

	return nil
}

func RunDatabaseMigrations(db *sql.DB) error {
	tables := []string{"queue", "donos"}
	for _, table := range tables {
		err := addColumnIfNotExist(db, table, "usd_amount", "FLOAT")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, table, "media_url", "TEXT")
		if err != nil {
			return err
		}
	}
	tables = []string{"users"}
	for _, table := range tables {
		err := addColumnIfNotExist(db, table, "links", "TEXT")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, table, "dono_gif", "TEXT")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, table, "default_crypto", "TEXT")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, table, "dono_sound", "TEXT")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, table, "alert_url", "TEXT")
		if err != nil {
			return err
		}

		err = removeColumnIfExist(db, table, "progressbar_url")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, "users", "date_enabled", "DATETIME")
		if err != nil {
			return err
		}

		err = addColumnIfNotExist(db, "users", "wallet_uploaded", "BOOLEAN")
		if err != nil {
			return err
		}
	}

	tables = []string{"queue"}
	for _, table := range tables {
		err := addColumnIfNotExist(db, table, "user_id", "TEXT")
		if err != nil {
			return err
		}
	}

	err := updateColumnAlertURLIfNull(db, "users", "alert_url")
	if err != nil {
		return err
	}

	err = updateColumnWalletUploadedIfNull(db, "users", "wallet_uploaded")
	if err != nil {
		return err
	}

	err = updateColumnDateEnabledIfNull(db, "users", "date_enabled")
	if err != nil {
		return err
	}

	return nil
}

func updateColumnWalletUploadedIfNull(db *sql.DB, tableName, columnName string) error {
	if checkDatabaseColumnExist(db, tableName, columnName) {
		_, err := db.Exec(`UPDATE `+tableName+` SET `+columnName+` = ? WHERE `+columnName+` IS NULL`, "0")
		if err != nil {
			return err
		}
	}
	return nil
}

func updateColumnDateEnabledIfNull(db *sql.DB, tableName, columnName string) error {
	if checkDatabaseColumnExist(db, tableName, columnName) {
		_, err := db.Exec(`UPDATE `+tableName+` SET `+columnName+` = ? WHERE `+columnName+` IS NULL`, time.Now().UTC())
		if err != nil {
			return err
		}
	}
	return nil
}

func updateColumnAlertURLIfNull(db *sql.DB, tableName, columnName string) error {
	if checkDatabaseColumnExist(db, tableName, columnName) {
		value := GenerateUniqueURL()
		_, err := db.Exec(`UPDATE `+tableName+` SET `+columnName+` = ? WHERE `+columnName+` IS NULL`, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeColumnIfExist(db *sql.DB, tableName, columnName string) error {
	if checkDatabaseColumnExist(db, tableName, columnName) {
		_, err := db.Exec(`ALTER TABLE ` + tableName + ` DROP COLUMN ` + columnName)
		if err != nil {
			return err
		}
	}
	return nil
}

func addColumnIfNotExist(db *sql.DB, tableName, columnName, columnType string) error {
	if !checkDatabaseColumnExist(db, tableName, columnName) {
		_, err := db.Exec(`ALTER TABLE ` + tableName + ` ADD COLUMN ` + columnName + ` ` + columnType)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkDatabaseColumnExist(db *sql.DB, dbTable string, dbColumn string) bool {
	// check if column already exists
	var count int
	err := db.QueryRow("SELECT count(*) FROM pragma_table_info('" + dbTable + "') WHERE name='" + dbColumn + "'").Scan(&count)
	if err != nil {
		return false
	}

	// column doesn't exist
	if count == 0 {
		return false
	}
	return true // column does exist
}

func CryptosStructToJSONString(s CryptosEnabled) string {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Println("cryptosStructToJSONString error:", err)
		return ""
	}
	return string(bytes)
}

func CryptosJsonStringToStruct(jsonStr string) CryptosEnabled {
	var s CryptosEnabled
	err := json.Unmarshal([]byte(jsonStr), &s)
	if err != nil {
		log.Println("cryptosJsonStringToStruct error:", err)
		return CryptosEnabled{}
	}
	return s
}

func CreateObsTable(db *sql.DB) error {
	obsTable := `
        CREATE TABLE IF NOT EXISTS obs (
            id INTEGER PRIMARY KEY,
            user_id INTEGER,
            gif_name TEXT,
            mp3_name TEXT,
            tts_voice TEXT,
            message TEXT,
            needed FLOAT,
            sent FLOAT
        );`
	_, err := db.Exec(obsTable)
	return err
}

func CompareStringsLowercase(str_one, str_two string) bool {
	if strings.ToLower(str_one) == strings.ToLower(str_two) {
		return true
	} else {
		return false
	}
}

func GenerateUniqueURL() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const length = 30
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return (string(randomString))
}

func StartWallets(ur *UserRepository, mr *MoneroRepository, sr *SolRepository) {
	ur.PrintUserColumns()
	users, err := ur.GetAll()
	if err != nil {
		log.Fatalf("startWallet() error: %v", err)
	}

	for _, user := range users {
		log.Println("Checking user:", user.Username, "User ID:", user.UserID, "User billing data enabled:", user.BillingData.Enabled)
		if user.BillingData.Enabled {
			log.Println("User valid", user.UserID, "User eth_address:", ur.Users[user.UserID].EthAddress)
			if user.WalletUploaded {
				log.Println("Monero wallet uploaded")
				mr.XmrWallets = append(mr.XmrWallets, []int{user.UserID, starting_port})
				go mr.StartMoneroWallet(starting_port, user.UserID, user)
				starting_port++

			} else {
				if ur.CheckWalletExists(user.UserID) {
					log.Println("Monero wallet uploaded")
					mr.XmrWallets = append(mr.XmrWallets, []int{user.UserID, starting_port})
					go mr.StartMoneroWallet(starting_port, user.UserID, user)
					user.WalletUploaded = true
					ur.Update(user)
					starting_port++
				} else {
					log.Println("Monero wallet not uploaded")
				}
			}
		} else {
			log.Println("startWallets() User not valid")
		}
	}

	fmt.Println("startWallet() starting monitoring of solana addresses.")
	for _, user := range users {
		sr.wallets[user.UserID] = SolWallet{
			Address: user.SolAddress,
			Amount:  0.00,
		}
	}

	sr.SetSolWallets(sr.wallets)
	go sr.StartMonitoringSolana()
}

func ReturnIPPenalty(ips []string, currentDonoIP string) float64 {
	// Check if the encrypted IP matches any of the encrypted IPs in the slice of donos
	sameIPCount := 0
	for _, donoIP := range ips {
		if donoIP == currentDonoIP {
			sameIPCount++
		}
	}
	// Calculate the exponential delay factor based on the number of matching IPs
	expoAdder := 1.00
	if sameIPCount > 2 {
		expoAdder = math.Pow(1.3, float64(sameIPCount)) / 1.3
	}
	return expoAdder
}

func FormatMediaURL(media_url string) string {
	isValid, timecode, properLink := isYouTubeLink(media_url)
	log.Println(isValid, timecode, properLink)

	embedLink := ""
	if isValid {
		videoID := ExtractVideoID(properLink)
		embedLink = fmt.Sprintf(videoID)
	}
	return embedLink
}

func isYouTubeLink(link string) (bool, int, string) {
	var timecode int
	var properLink string

	youtubeRegex := regexp.MustCompile(`^(?:https?://)?(?:www\.)?(?:youtube\.com/watch\?v=|youtu\.be/)([^&]+)(?:\?t=)?(\d*)$`)
	embedRegex := regexp.MustCompile(`^(?:https?://)?(?:www\.)?youtube\.com/embed/([^?]+)(?:\?start=)?(\d*)$`)

	if youtubeMatches := youtubeRegex.FindStringSubmatch(link); youtubeMatches != nil {
		if len(youtubeMatches[2]) > 0 {
			fmt.Sscanf(youtubeMatches[2], "%d", &timecode)
		}
		properLink = "https://www.youtube.com/watch?v=" + youtubeMatches[1]
		return true, timecode, properLink
	}

	if embedMatches := embedRegex.FindStringSubmatch(link); embedMatches != nil {
		if len(embedMatches[2]) > 0 {
			fmt.Sscanf(embedMatches[2], "%d", &timecode)
		}
		properLink = "https://www.youtube.com/watch?v=" + embedMatches[1]
		return true, timecode, properLink
	}

	return false, 0, ""
}

// extractVideoID extracts the video ID from a YouTube URL
func ExtractVideoID(url string) string {
	videoID := ""
	// Use a regular expression to extract the video ID from the YouTube URL
	re := regexp.MustCompile(`v=([\w-]+)`)
	match := re.FindStringSubmatch(url)
	if len(match) == 2 {
		videoID = match[1]
	}
	return videoID
}

func CheckDonoForMediaUSDThreshold(media_url string, dono_usd float64) (bool, string) {
	valid := true
	if dono_usd < float64(ServerMinMediaDono) {
		media_url = ""
		valid = false

	}
	return valid, media_url
}

func ConvertToFloat64(value string) float64 {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return f
}

func FetchExchangeRates(ur *UserRepository) {
	for {
		// Fetch the exchange rate data from the API
		var err error
		prices, err = GetCryptoPrices()
		if err != nil {
			log.Println(err)
		} else {
			ur.SetMinDonos()
		}

		time.Sleep(80 * time.Second)
	}

}

func GetCryptoPrices() (CryptoPrice, error) {

	// Call the Coingecko API to get the current price for each cryptocurrency
	url := "https://api.coingecko.com/api/v3/simple/price?ids=monero,solana,ethereum,paint,hex,matic-network,binance-usd,shiba-inu,kleros&vs_currencies=usd"
	resp, err := http.Get(url)
	if err != nil {
		return prices, err
	}
	defer resp.Body.Close()

	var data map[string]map[string]float64
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return prices, err
	}

	prices = CryptoPrice{
		Monero:     data["monero"]["usd"],
		Solana:     data["solana"]["usd"],
		Ethereum:   data["ethereum"]["usd"],
		Paint:      data["paint"]["usd"],
		Hexcoin:    data["hex"]["usd"],
		Polygon:    data["matic-network"]["usd"],
		BinanceUSD: data["binance-usd"]["usd"],
		ShibaInu:   data["shiba-inu"]["usd"],
		Kleros:     data["kleros"]["usd"],
	}

	return prices, nil
}

func CheckPendingAccounts(dr *DonoRepository) {
	for {

		for _, transaction := range dr.Transfers {
			tN := dr.GetTransactionToken(transaction)
			if tN == "ETH" && transaction.To == dr.UserRepo.GetAdminETHAdd() {
				valueStr := fmt.Sprintf("%.18f", transaction.Value)

				for _, user := range dr.UserRepo.PendingUsers {
					if user.ETHNeeded == valueStr {
						err := dr.UserRepo.CreateNewUserFromPending(user)
						if err != nil {
							log.Println("Error marking payment as complete:", err)
						} else {
							log.Println("Payment marked as complete for:", user.Username)
						}
					}
				}
			}
		}

		for _, user := range dr.UserRepo.PendingUsers {
			xmrFl, _ := dr.MoneroRepo.getBalance(user.XMRPayID, 1)
			xmrSent, _ := StandardizeFloatToString(xmrFl)

			log.Println("XMR sent:", xmrSent)
			xmrSentStr, _ := ConvertStringTo18DecimalPlaces(xmrSent)
			log.Println("XMR sent str:", xmrSentStr)
			log.Println("XMRNeeded str:", user.XMRNeeded)
			if user.XMRNeeded == xmrSentStr {
				err := dr.UserRepo.CreateNewUserFromPending(user)
				if err != nil {
					log.Println("Error marking payment as complete:", err)
				} else {
					log.Println("Payment marked as complete for:", user.Username)
				}
			}
		}

		time.Sleep(time.Duration(25) * time.Second)
	}
}

func CheckBillingAccounts(dr *DonoRepository) {
	for {
		tMapGenerated := false
		transactionMap := make(map[string]Transfer)

		for _, user := range dr.UserRepo.Users {

			if user.BillingData.NeedToPay {

				xmrFl, _ := dr.MoneroRepo.getBalance(user.BillingData.XMRPayID, 1)
				xmrSent, _ := StandardizeFloatToString(xmrFl)
				xmrSentStr, _ := StandardizeString(xmrSent)
				if user.BillingData.XMRAmount == xmrSentStr {
					dr.UserRepo.RenewUserSubscription(user)
					continue
				}

				adminETHAdd := dr.UserRepo.GetAdminETHAdd()

				if !tMapGenerated { //Generate Map from transaction slice
					for _, transaction := range dr.Transfers {
						hash := dr.GetTransactionAmount(transaction)
						standard_hash, _ := StandardizeString(hash)
						dr.TransactionMap[standard_hash] = transaction
					}
					tMapGenerated = true
				}

				valueToCheck, _ := StandardizeString(user.BillingData.ETHAmount)
				transaction, ok := transactionMap[valueToCheck]
				if ok {
					tN := dr.GetTransactionToken(transaction)
					if tN == "ETH" && transaction.To == adminETHAdd {
						dr.UserRepo.RenewUserSubscription(user)
						continue
					}
				}
			}
		}
		time.Sleep(time.Duration(30) * time.Second)
	}
}

func GetUSDValue(as float64, c string) float64 {
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

func SetServerVars(ur *UserRepository) {
	log.Println("Starting.")
	log.Println("		 ..")
	time.Sleep(2 * time.Second)
	log.Println("------------ setServerVars()")
	ur.SetMinDonos()
}

func GetNewAccountXMRPrice() string {
	xmrPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", (15.00/prices.Monero)), 64)
	xmrStr, _ := StandardizeFloatToString(xmrPrice)
	return xmrStr
}

func GetXMRAmountInUSD(usdAmount float64) string {
	xmrPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", (usdAmount/prices.Monero)), 64)
	xmrStr, _ := StandardizeFloatToString(xmrPrice)
	return xmrStr
}

func GetETHAmountInUSD(usdAmount float64) string {
	ethPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.18f", (usdAmount/prices.Ethereum)), 64)
	ethStr := FuzzDono(ethPrice, "ETH")
	ethStr_, _ := StandardizeFloatToString(ethStr)
	return ethStr_
}

func GetNewAccountETHPrice() string {
	ethPrice, _ := strconv.ParseFloat(fmt.Sprintf("%.18f", (15.00/prices.Ethereum)), 64)
	ethStr := FuzzDono(ethPrice, "ETH")
	ethStr_, _ := StandardizeFloatToString(ethStr)
	return ethStr_
}

func CreateNewPendingUser(username string, password string, dr *DonoRepository, ur *UserRepository, mr *MoneroRepository) (PendingUser, error) {
	log.Println("begin createNewPendingUser()")
	user_, _ := ur.GetUserByUsernameCached("admin")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return PendingUser{}, err
	}
	PayID, PayAddress := mr.GetNewAccountXMR()
	user := PendingUser{
		Username:       username,
		HashedPassword: hashedPassword,
		ETHAddress:     user_.EthAddress,
		XMRAddress:     PayAddress,
		ETHNeeded:      GetNewAccountETHPrice(),
		XMRNeeded:      GetNewAccountXMRPrice(),
		XMRPayID:       PayID,
	}

	err = ur.CreatePending(user)
	if err != nil {
		log.Println("createPendingUser:", err)
		return PendingUser{}, err
	}
	// Get the ID of the newly inserted user
	row := ur.Db.QueryRow(`SELECT last_insert_rowid()`)
	err = row.Scan(&user.ID)
	if err != nil {
		return PendingUser{}, err
	}
	ur.PendingUsers[user.ID] = user
	log.Println("finish createNewPendingUser() without error")
	return user, nil

}

func CreatePendingDono(name string, message string, mediaURL string, amountNeeded float64, cryptoCode string, encrypted_ip string) SuperChat {
	amountNeeded = FuzzDono(amountNeeded, cryptoCode)
	pendingDono := SuperChat{
		Name:         name,
		Message:      message,
		MediaURL:     mediaURL,
		AmountNeeded: amountNeeded,
		Completed:    false,
		CreatedAt:    time.Now().String(),
		CheckedAt:    time.Now().String(),
		CryptoCode:   cryptoCode,
		EncryptedIP:  encrypted_ip,
	}
	return pendingDono
}

func AppendPendingDono(pending_donos []SuperChat, new_dono SuperChat) []SuperChat {
	pending_donos = append(pending_donos, new_dono)
	return pending_donos
}

func EthToWei(ethStr string) *big.Int {
	etherValue := big.NewFloat(1000000000000000000)
	f, err := strconv.ParseFloat(ethStr, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	number := big.NewFloat(f)

	weiValue := new(big.Int)
	weiValue, _ = weiValue.SetString(number.Mul(number, etherValue).Text('f', 0), 10)

	return weiValue
}

func GetIPAddress(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}

	if ip == "" {
		ip = r.RemoteAddr
	}

	return ip
}

func CondenseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TruncateStrings(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for !utf8.ValidString(s[:n]) {
		n--
	}
	return s[:n]
}

func GetUserPathByID(id int) string {
	return fmt.Sprintf("users/%d/", id)
}

func CheckFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		// File exists
		return true
	} else {
		return false
	}

}

func CheckUserGIF(userpath string) bool {
	up := userpath + "gifs/default.gif"
	//log.Println("checking", up)
	b := CheckFileExists(up)
	if b {
		log.Println("user gif exists")
	} else {
		log.Println("user gif doesn't exist")
	}
	return b
}

func CheckUserSound(userpath string) bool {
	up := userpath + "sounds/default.mp3"
	//log.Println("checking", up)
	b := CheckFileExists(up)
	if b {
		log.Println("user sound exists")
	} else {
		log.Println("user sound doesn't exist")
	}
	return b
}

func CheckUserMoneroWallet(userpath string) bool {
	up := userpath + "monero/wallet"
	//log.Println("checking", up)
	b := CheckFileExists(up)
	if b {
		log.Println("user wallet exists")
	} else {
		log.Println("user wallet doesn't exist")
	}
	return b
}

func CheckUserMoneroWalletKeys(userpath string) bool {
	up := userpath + "monero/wallet"
	//log.Println("checking", up)
	b := CheckFileExists(up)
	if b {
		log.Println("user wallet keys exists")
	} else {
		log.Println("user wallet keys doesn't exist")
	}
	return b
}

func SaveFileToDisk(file multipart.File, header *multipart.FileHeader, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return err
	}

	return nil
}

func GetCryptoDecimalsByCode(code string) (int, error) {
	if code == "ETH" {
		return 18, nil
	} else {
		code = strings.ToUpper(code)
		for _, cryptoInfo := range cryptoMap {
			if cryptoInfo["code"] == code {
				decimals, ok := cryptoInfo["decimals"].(int)
				if !ok {
					return 0, fmt.Errorf("decimals value for crypto with code %s is not an integer", code)
				}
				return decimals, nil
			}
		}
		return 0, fmt.Errorf("crypto with code %s not found", code)
	}
}

func GetCryptoContractByCode(code string) (string, error) {
	code = strings.ToUpper(code)
	for _, cryptoInfo := range cryptoMap {
		if cryptoInfo["code"] == code {
			contract, ok := cryptoInfo["contract"].(string)
			if !ok {
				return "", fmt.Errorf("contract value for %s is not a string", code)
			}
			return contract, nil
		}
	}
	return "", fmt.Errorf("crypto with code %s not found", code)
}

func CheckValidSubscription(DateEnabled time.Time) bool {
	oneMonthAhead := DateEnabled.AddDate(0, 1, 0)
	if oneMonthAhead.After(time.Now().UTC()) {
		log.Println("User valid")
		return true
	}
	log.Println("checkValidSubscription() User not valid")
	return false
}
