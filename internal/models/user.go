package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	//	"github.com/davecgh/go-spew/spew"

	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var amountNeeded = 1000.00
var amountSent = 200.00
var minDonoValue float64 = 5.0

const username = "admin"

type User struct {
	UserID               int
	Username             string
	HashedPassword       []byte
	EthAddress           string
	SolAddress           string
	HexcoinAddress       string
	XMRWalletPassword    string
	MinDono              int
	MinMediaDono         int
	MediaEnabled         bool
	CreationDatetime     string
	ModificationDatetime string
	Links                string
	DonoGIF              string
	DonoSound            string
	AlertURL             string
	MinSol               float64
	MinEth               float64
	MinXmr               float64
	MinPaint             float64
	MinHex               float64
	MinMatic             float64
	MinBusd              float64
	MinShib              float64
	MinUsdc              float64
	MinTusd              float64
	MinWbtc              float64
	MinPnk               float64
	DateEnabled          time.Time
	WalletUploaded       bool
	WalletPending        bool
	CryptosEnabled       CryptosEnabled
	BillingData          BillingData
	DefaultCrypto        string
}

type UserPageData struct {
	ErrorMessage string
}

// Add the following struct to store the incoming data
type UpdateCryptosRequest struct {
	UserID          string          `json:"userId"`
	SelectedCryptos map[string]bool `json:"selectedCryptos"`
}

type AccountPayData struct {
	Username    string
	AmountXMR   string
	AmountETH   string
	AddressXMR  string
	AddressETH  string
	QRB64XMR    string
	QRB64ETH    string
	UserID      int
	BillingData BillingData
	DateCreated time.Time
}

type Link struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type PendingUser struct {
	ID             int
	Username       string
	HashedPassword []byte
	XMRPayID       string
	ETHNeeded      string
	XMRNeeded      string
	ETHAddress     string
	XMRAddress     string
}

type CryptosEnabled struct {
	XMR   bool
	SOL   bool
	ETH   bool
	PAINT bool
	HEX   bool
	MATIC bool
	BUSD  bool
	SHIB  bool
	PNK   bool
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

type UserRepositoryInterface interface {
	GetByID(userID int) (User, error)
	CreateAdmin()
	GetNew(username string, hashedPassword []byte) User
	CreateNew(username, password string) error
	Create(user User) int
	GetAll() ([]User, error)
	Update(user User) error
	UpdateObsData(userID int, gifName string, mp3Name string, ttsVoice string, pbData ProgressbarData) error
	CreateNewOBS(userID int, message string, needed, sent float64, refresh int, gifFile, soundFile, ttsVoice string)
	InsertObsData(userId int, gifName, mp3Name, ttsVoice string, pbData ProgressbarData) error
	GetOBSDataByUserID(userID int) (OBSDataStruct, error)
	PrintUserColumns() error
	SetUserMinDonos(user User) User
	SetMinDonos()
	GetAdminETHAdd() string
	GetUserBySessionCached(sessionToken string) (User, bool)
	GetUserByUsernameCached(username string) (User, bool)
	GetUserByUsername(username string) (User, error)
	CreateNewUserFromPending(user_ PendingUser) error
	DeletePendingUser(user PendingUser) error
	RenewUserSubscription(user User)
	GetObsData(userId int) OBSDataStruct
	VerifyPassword(user User, password string) bool
	GetEthAddressByID(userID int) string
	GetSolAddressByID(userID int) string
	CreatePending(user PendingUser) error
	GetOBSDataByAlertURL(AlertURL string) (OBSDataStruct, error)
	GetUserByAlertURL(AlertURL string) (User, error)
	CheckUserByID(id int) bool
	CheckUserByUsername(username string) (bool, int)
	GetUserBySession(sessionToken string) (User, error)
	GetUserLinks(user User) ([]Link, error)
	GetUserCryptosEnabled(user User) (User, error)
	GetActiveXMRUsers() ([]*User, error)
	GetActiveETHUsers() ([]*User, error)
	UpdateEnabledDate(userID int) error
	MapToCryptosEnabled(selectedCryptos map[string]bool) CryptosEnabled
	CreateSession(userID int) (string, error)
	ValidateSession(r *http.Request) (int, error)
}

type UserRepository struct {
	Db           *sql.DB
	SolRepo      *SolRepository
	BillingRepo  *BillingRepository
	InviteRepo   *InviteRepository
	Users        map[int]User
	PendingUsers map[int]PendingUser
	UserSessions map[string]int
}

func NewUserRepository(db *sql.DB, sr *SolRepository, ir *InviteRepository, br *BillingRepository) *UserRepository {
	return &UserRepository{
		Db:           db,
		SolRepo:      sr,
		BillingRepo:  br,
		InviteRepo:   ir,
		Users:        make(map[int]User),
		PendingUsers: make(map[int]PendingUser),
		UserSessions: make(map[string]int),
	}
}

func (ur *UserRepository) GetByID(userID int) (User, error) {
	user, ok := ur.Users[userID]
	if !ok {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func (ur *UserRepository) CreateAdmin() {

	ur.CreateNew("admin", "hunter123")
}

func (ur *UserRepository) GetNew(username string, hashedPassword []byte) User {

	ce := CryptosEnabled{
		XMR:   false,
		SOL:   false,
		ETH:   false,
		PAINT: false,
		HEX:   false,
		MATIC: false,
		BUSD:  false,
		SHIB:  false,
		PNK:   false,
	}

	user := User{
		Username:          username,
		HashedPassword:    hashedPassword,
		CryptosEnabled:    ce,
		EthAddress:        "0x5b5856dA280e592e166A1634d353A53224ed409c",
		SolAddress:        "adWqokePHcAbyF11TgfvvM1eKax3Kxtnn9sZVQh6fXo",
		HexcoinAddress:    "0x5b5856dA280e592e166A1634d353A53224ed409c",
		XMRWalletPassword: "",
		MinDono:           3,
		MinMediaDono:      5,
		MediaEnabled:      true,
		DonoGIF:           "default.gif",
		DonoSound:         "default.mp3",
		AlertURL:          GenerateUniqueURL(),
		WalletUploaded:    false,
		Links:             "",
		DateEnabled:       time.Now().UTC(),
	}
	return user
}

func (ur *UserRepository) CreateNew(username, password string) error {
	log.Println("running createNewUser")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		return err
	}
	// create admin user if not exists
	user := ur.GetNew(username, hashedPassword)
	userID := ur.Create(user)
	if userID != 0 {
		ur.CreateNewOBS(userID, "default message", 100.00, 50.00, 5, user.DonoGIF, user.DonoSound, "test_voice")
		log.Println("createUser() succeeded, so OBS row was created.")
	} else {
		log.Println("createUser() didn't succeed, so OBS row wasn't created.")
	}

	log.Println("finished createNewUser")
	return nil
}

func (ur *UserRepository) Create(user User) int {
	log.Println("running CreateUser")
	// Insert the user's data into the database

	ce := CryptosEnabled{
		XMR:   false,
		SOL:   true,
		ETH:   true,
		PAINT: true,
		HEX:   true,
		MATIC: true,
		BUSD:  true,
		SHIB:  true,
		PNK:   true,
	}

	ce_ := CryptosStructToJSONString(ce)

	_, err := ur.Db.Exec(`
        INSERT INTO users (
            username,
            HashedPassword,
            eth_address,
            sol_address,
            hex_address,
            xmr_wallet_password,
            min_donation_threshold,
            min_media_threshold,
            media_enabled,
            created_at,
            modified_at,
            links,
            dono_gif,
            dono_sound,
            alert_url,
            date_enabled,
            wallet_uploaded,
            cryptos_enabled,
            default_crypto
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, user.Username, user.HashedPassword, user.EthAddress, user.SolAddress, user.HexcoinAddress, "", user.MinDono, user.MinMediaDono, user.MediaEnabled, time.Now().UTC(), time.Now(), "", user.DonoGIF, user.DonoSound, user.AlertURL, user.DateEnabled, 0, ce_, user.DefaultCrypto)

	if err != nil {
		log.Println(err)
		return 0
	}

	// Get the ID of the newly created user
	row := ur.Db.QueryRow(`SELECT last_insert_rowid()`)
	var userID int
	err = row.Scan(&userID)
	if err != nil {
		log.Println(err)
		return 0
	}

	billing := BillingData{
		UserID:          userID,
		AmountThisMonth: 0.00,
		AmountTotal:     0.00,
		Enabled:         true,
		NeedToPay:       false,
		ETHAmount:       "",
		XMRAmount:       "",
		XMRPayID:        "",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	_, err = ur.Db.Exec(`
        INSERT INTO billing (
            user_id,
            amount_this_month,
            amount_total,
            enabled,
            need_to_pay,
            eth_amount,
            xmr_amount,
            xmr_pay_id,
            created_at,
            updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, billing.UserID, billing.AmountThisMonth, billing.AmountTotal, billing.Enabled, billing.NeedToPay, "", "", "", billing.CreatedAt, billing.CreatedAt)

	user.BillingData = billing
	ur.Users[userID] = user

	log.Printf("BillingData.Enabled: %v", billing.Enabled)

	// Create a directory for the user based on their ID
	userDir := fmt.Sprintf("users/%d", userID)
	err = os.MkdirAll(userDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}

	// Create "gifs" and "sounds" subfolders inside the user's directory
	gifsDir := fmt.Sprintf("%s/gifs", userDir)
	err = os.MkdirAll(gifsDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}

	soundsDir := fmt.Sprintf("%s/sounds", userDir)
	err = os.MkdirAll(soundsDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}

	moneroDir := fmt.Sprintf("%s/monero", userDir)
	err = os.MkdirAll(moneroDir, os.ModePerm)
	if err != nil {
		log.Println(err)
	}

	minDonoValue = float64(user.MinDono)
	log.Println("finished createNewUser")

	_, err = ur.GetAll()
	if err != nil {
		log.Fatalf("createUser() getAllUsers() error: %v", err)
	}

	return userID
}

func (ur *UserRepository) GetAll() ([]User, error) {
	var users []User
	rows, err := ur.Db.Query("SELECT * FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		var links, donoGIF, donoSound, alertURL, defaultCrypto, cryptosEnabled sql.NullString

		err = rows.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress,
			&user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono,
			&user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &links, &donoGIF, &donoSound,
			&alertURL, &user.DateEnabled, &user.WalletUploaded, &cryptosEnabled, &defaultCrypto)

		if err != nil {
			return nil, err
		}

		user.Links = links.String
		if !links.Valid {
			user.Links = ""
		}

		user.DonoGIF = donoGIF.String
		if !donoGIF.Valid {
			user.DonoGIF = "default.gif"
		}

		user.DonoSound = donoSound.String
		if !donoSound.Valid {
			user.DonoSound = "default.mp3"
		}

		user.DefaultCrypto = defaultCrypto.String
		if !defaultCrypto.Valid {
			user.DefaultCrypto = ""
		}

		user.AlertURL = alertURL.String
		if !alertURL.Valid {
			user.AlertURL = GenerateUniqueURL()
		}

		ce := CryptosEnabled{
			XMR:   true,
			SOL:   true,
			ETH:   false,
			PAINT: false,
			HEX:   true,
			MATIC: false,
			BUSD:  true,
			SHIB:  false,
			PNK:   true,
		}

		user.CryptosEnabled = CryptosJsonStringToStruct(cryptosEnabled.String)
		if !cryptosEnabled.Valid {
			log.Println("user cryptos enabled not fixed")
			user.CryptosEnabled = ce
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	billings, err := ur.BillingRepo.getAllBilling()
	if err != nil {
		return nil, err
	}

	for _, billing := range billings {
		ur.BillingRepo.billings[billing.UserID] = billing
	}

	for i := range users {
		billing, ok := ur.BillingRepo.billings[users[i].UserID]
		if ok {
			users[i].BillingData = billing
			ur.Users[users[i].UserID] = users[i]
		}
	}

	return users, nil
}

// old: updateUser
func (ur *UserRepository) Update(user User) error {
	ur.Users[user.UserID] = user
	statement := `
		UPDATE users
		SET Username=?, HashedPassword=?, eth_address=?, sol_address=?, hex_address=?,
			xmr_wallet_password=?, min_donation_threshold=?, min_media_threshold=?, media_enabled=?, modified_at=?, links=?, dono_gif=?, dono_sound=?, alert_url=?, date_enabled=?, wallet_uploaded=?, cryptos_enabled=?, default_crypto=?
		WHERE id=?
	`
	_, err := ur.Db.Exec(statement, user.Username, user.HashedPassword, user.EthAddress,
		user.SolAddress, user.HexcoinAddress, user.XMRWalletPassword, user.MinDono, user.MinMediaDono,
		user.MediaEnabled, time.Now().UTC(), user.Links, user.DonoGIF, user.DonoSound, user.AlertURL, user.DateEnabled, user.WalletUploaded, CryptosStructToJSONString(user.CryptosEnabled), user.DefaultCrypto, user.UserID)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}

	statement = `
		UPDATE billing
		SET user_id=?, amount_this_month=?, amount_total=?, enabled=?, need_to_pay=?,
			eth_amount=?, xmr_amount=?, xmr_pay_id=?, created_at=?, updated_at=?
		WHERE billing_id=?
	`
	_, err = ur.Db.Exec(statement, user.UserID, user.BillingData.AmountThisMonth, user.BillingData.AmountTotal, user.BillingData.Enabled,
		user.BillingData.NeedToPay, user.BillingData.ETHAmount, user.BillingData.XMRAmount, user.BillingData.XMRPayID, user.BillingData.CreatedAt,
		user.BillingData.UpdatedAt, user.BillingData.UserID)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}

	ur.SolRepo.wallets[user.UserID] = SolWallet{
		Address: user.SolAddress,
		Amount:  0.00,
	}

	ur.SolRepo.SetSolWallets(ur.SolRepo.wallets)
	return err
}

func (ur *UserRepository) UpdateObsData(userID int, gifName string, mp3Name string, ttsVoice string, pbData ProgressbarData) error {
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
	_, err := ur.Db.Exec(updateObsData, userID, gifName, mp3Name, ttsVoice, pbData.Message, pbData.Needed, pbData.Sent, userID)
	return err
}

func (ur *UserRepository) CreateNewOBS(userID int, message string, needed, sent float64, refresh int, gifFile, soundFile, ttsVoice string) {
	pbData := ProgressbarData{
		Message: message,
		Needed:  needed,
		Sent:    sent,
		Refresh: refresh,
	}
	err := ur.InsertObsData(userID, gifFile, soundFile, ttsVoice, pbData)
	if err != nil {
		log.Fatal(err)
	}

}

func (ur *UserRepository) InsertObsData(userId int, gifName, mp3Name, ttsVoice string, pbData ProgressbarData) error {
	obsData := `
        INSERT INTO obs (
            user_id,
            gif_name,
            mp3_name,
            tts_voice,
            message,
            needed,
            sent
        ) VALUES (?, ?, ?, ?, ?, ?, ?);`
	_, err := ur.Db.Exec(obsData, userId, gifName, mp3Name, ttsVoice, pbData.Message, pbData.Needed, pbData.Sent)
	return err
}

func (ur *UserRepository) GetOBSDataByUserID(userID int) (OBSDataStruct, error) {
	var obsData OBSDataStruct
	//var alertURL sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.Db.QueryRow("SELECT gif_name, mp3_name, `message`, needed, sent FROM obs WHERE user_id=?", userID)

	err := row.Scan(&obsData.FilenameGIF, &obsData.FilenameMP3, &obsData.Message, &obsData.Needed, &obsData.Sent)
	if err != nil {
		log.Println("Couldn't get obsData,", err)
		return obsData, err
	}

	return obsData, nil
}

func (ur *UserRepository) PrintUserColumns() error {
	rows, err := ur.Db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'users';`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var column string
	for rows.Next() {
		err = rows.Scan(&column)
		if err != nil {
			return err
		}
		fmt.Println(column)
	}
	return rows.Err()
}

func (ur *UserRepository) SetUserMinDonos(user User) User {
	var err error
	user.MinSol, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Solana)), 64)
	user.MinEth, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Ethereum)), 64)
	user.MinXmr, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Monero)), 64)
	user.MinPaint, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Paint)), 64)
	user.MinHex, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Hexcoin)), 64)
	user.MinMatic, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Polygon)), 64)
	user.MinBusd, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.BinanceUSD)), 64)
	user.MinShib, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.ShibaInu)), 64)
	user.MinUsdc, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono))), 64)
	user.MinTusd, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono))), 64)
	user.MinWbtc, _ = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.WBTC)), 64)
	user.MinPnk, err = strconv.ParseFloat(fmt.Sprintf("%.5f", (float64(user.MinDono)/prices.Kleros)), 64)
	if err != nil {
		log.Println("setUserMinDonos() err:", err)
	}

	return user
}

func (ur *UserRepository) SetMinDonos() {
	for i := range ur.Users {
		ur.Users[i] = ur.SetUserMinDonos(ur.Users[i])
	}
}

func (ur *UserRepository) GetAdminETHAdd() string {
	user, validUser := ur.GetUserByUsernameCached(username)

	if !validUser {
		return ""
	}

	return user.EthAddress
}

// get a user by their session token
func (ur *UserRepository) GetUserBySessionCached(sessionToken string) (User, bool) {
	userID, ok := ur.UserSessions[sessionToken]
	if !ok {
		log.Println("session token not found")
		return ur.Users[0], false
	}
	for _, user := range ur.Users {
		if user.UserID == userID {
			return user, true
		}
	}
	return ur.Users[0], false
}

func (ur *UserRepository) GetUserByUsernameCached(username string) (User, bool) {
	for _, user := range ur.Users {
		if strings.ToLower(user.Username) == strings.ToLower(username) {
			return user, true
		}
	}
	return ur.Users[0], false

}

// get a user by their username
// func (ur *UserRepository) GetUserByUsername(username string) (User, error) {
// 	var user User
// 	var links, donoGIF, defaultCrypto, donoSound, alertURL, cryptosEnabled sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
// 	row := ur.Db.QueryRow("SELECT * FROM users WHERE Username=?", username)
// 	err := row.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress,
// 		&user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono,
// 		&user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &links, &donoGIF, &donoSound, &alertURL, &user.DateEnabled, &user.WalletUploaded, &cryptosEnabled, &defaultCrypto)
// 	if err != nil {
// 		return User{}, err
// 	}
// 	user.Links = links.String
// 	if !links.Valid {
// 		user.Links = ""
// 	}
// 	user.DonoGIF = donoGIF.String // assign the sql.NullString to the user's "DonoGIF" field
// 	if !donoGIF.Valid {           // check if the "dono_gif" column is null
// 		user.DonoGIF = "default.gif" // set the user's "DonoGIF"
// 	}
// 	user.DonoSound = donoSound.String // assign the sql.NullString to the user's "DonoGIF" field
// 	if !donoSound.Valid {             // check if the "dono_gif" column is null
// 		user.DonoSound = "default.mp3" // set the user's "DonoSound"
// 	}

// 	user.DefaultCrypto = defaultCrypto.String // assign the sql.NullString to the user's "DonoGIF" field
// 	if !defaultCrypto.Valid {                 // check if the "dono_gif" column is null
// 		user.DefaultCrypto = "" // set the user's "DonoSound"
// 	}

// 	user.AlertURL = alertURL.String // assign the sql.NullString to the user's "DonoGIF" field
// 	if !alertURL.Valid {            // check if the "dono_gif" column is null
// 		user.AlertURL = GenerateUniqueURL() // set the user's "DonoSound"
// 	}

// 	ce := CryptosEnabled{
// 		XMR:   true,
// 		SOL:   true,
// 		ETH:   false,
// 		PAINT: false,
// 		HEX:   true,
// 		MATIC: false,
// 		BUSD:  true,
// 		SHIB:  false,
// 		PNK:   true,
// 	}

// 	user.CryptosEnabled = CryptosJsonStringToStruct(cryptosEnabled.String) // assign the sql.NullString to the user's "DonoGIF" field
// 	if !cryptosEnabled.Valid {                                             // check if the "dono_gif" column is null
// 		user.CryptosEnabled = ce // set the user's "DonoSound"
// 	}

// 	user = ur.SetUserMinDonos(user)

// 	return user, nil

// }

func (ur *UserRepository) CreateNewUserFromPending(user_ PendingUser) error {
	log.Println("running createNewUserFromPending")

	user := ur.GetNew(user_.Username, user_.HashedPassword)
	userID := ur.Create(user)
	if userID != 0 {
		ur.CreateNewOBS(userID, "default message", 100.00, 50.00, 5, user.DonoGIF, user.DonoSound, "test_voice")
		log.Println("createNewUserFromPending() succeeded, so OBS row was created. Deleting pending user from pendingusers table")
		err := ur.DeletePendingUser(user_)
		if err != nil {
			return err
		}
	} else {
		log.Println("createNewUserFromPending() didn't succeed, so OBS row wasn't created. Pending user remains in DB")
	}

	log.Println("finished createNewUserFromPending()")

	return nil
}

func (ur *UserRepository) DeletePendingUser(user PendingUser) error {
	delete(ur.PendingUsers, user.ID)
	_, err := ur.Db.Exec(`DELETE FROM pendingusers WHERE id = ?`, user.ID)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepository) RenewUserSubscription(user User) {
	user.BillingData.Enabled = true
	user.BillingData.AmountThisMonth = 0.00
	user.BillingData.AmountNeeded = 0.00
	user.BillingData.NeedToPay = false
	user.BillingData.UpdatedAt = time.Now().UTC()
	ur.Update(user)
}

func (ur *UserRepository) GetObsData(userId int) OBSDataStruct {
	var tempObsData OBSDataStruct
	err := ur.Db.QueryRow("SELECT gif_name, mp3_name, `message`, needed, sent FROM obs WHERE user_id = ?", userId).
		Scan(&tempObsData.FilenameGIF, &tempObsData.FilenameMP3, &tempObsData.Message, &tempObsData.Needed, &tempObsData.Sent)
	if err != nil {
		log.Println("Error:", err)
	}

	return tempObsData
}

// verify that the entered password matches the stored hashed password for a user
func (ur *UserRepository) VerifyPassword(user User, password string) bool {
	err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
	return err == nil
}

func (ur *UserRepository) GetEthAddressByID(userID int) string {
	if user, ok := ur.Users[userID]; ok {
		log.Println("Got userID:", user.UserID, "Returned:", user.EthAddress)
		return user.EthAddress
	}
	log.Println("Got userID:", userID, "No user found")
	return ""
}

func (ur *UserRepository) GetSolAddressByID(userID int) string {
	if user, ok := ur.Users[userID]; ok {
		log.Println("Got userID:", user.UserID, "Returned:", user.SolAddress)
		return user.SolAddress
	}
	log.Println("Got userID:", userID, "No user found")
	return ""
}

func (ur *UserRepository) CreatePending(user PendingUser) error {
	_, err := ur.Db.Exec(`
        INSERT INTO pendingusers (username, HashedPassword, XMRPayID, XMRNeeded, ETHNeeded)
        VALUES (?, ?, ?, ?, ?)
    `, user.Username, user.HashedPassword, user.XMRPayID, user.XMRNeeded, user.ETHNeeded)
	if err != nil {
		return err
	}

	// Get the ID of the newly inserted user
	row := ur.Db.QueryRow(`SELECT last_insert_rowid()`)
	err = row.Scan(&user.ID)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepository) GetOBSDataByAlertURL(AlertURL string) (OBSDataStruct, error) {
	user, err := ur.GetUserByAlertURL(AlertURL)
	if err != nil {
		log.Println("Couldn't get user,", err)
	}
	var obsData OBSDataStruct
	//var alertURL sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.Db.QueryRow("SELECT gif_name, mp3_name, `message`, needed, sent FROM obs WHERE user_id=?", user.UserID)

	err = row.Scan(&obsData.FilenameGIF, &obsData.FilenameMP3, &obsData.Message, &obsData.Needed, &obsData.Sent)
	if err != nil {
		log.Println("Couldn't get obsData,", err)
		return obsData, err
	}

	return obsData, nil

}

func (ur *UserRepository) GetUserByAlertURL(AlertURL string) (User, error) {
	var user User
	var links, donoGIF, defaultCrypto, donoSound, alertURL, cryptosEnabled sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.Db.QueryRow("SELECT * FROM users WHERE alert_url=?", AlertURL)
	err := row.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress,
		&user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono,
		&user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &links, &donoGIF, &donoSound, &alertURL, &user.DateEnabled, &user.WalletUploaded, &cryptosEnabled, &defaultCrypto)
	if err != nil {
		return User{}, err
	}
	user.Links = links.String
	if !links.Valid {
		user.Links = ""
	}
	user.DonoGIF = donoGIF.String // assign the sql.NullString to the user's "DonoGIF" field
	if !donoGIF.Valid {           // check if the "dono_gif" column is null
		user.DonoGIF = "default.gif" // set the user's "DonoGIF"
	}
	user.DonoSound = donoSound.String // assign the sql.NullString to the user's "DonoGIF" field
	if !donoSound.Valid {             // check if the "dono_gif" column is null
		user.DonoSound = "default.mp3" // set the user's "DonoSound"
	}

	user.DefaultCrypto = defaultCrypto.String // assign the sql.NullString to the user's "DonoGIF" field
	if !defaultCrypto.Valid {                 // check if the "dono_gif" column is null
		user.DefaultCrypto = "" // set the user's "DonoSound"
	}

	user.AlertURL = alertURL.String // assign the sql.NullString to the user's "DonoGIF" field
	if !alertURL.Valid {            // check if the "dono_gif" column is null
		user.AlertURL = GenerateUniqueURL() // set the user's "DonoSound"
	}

	ce := CryptosEnabled{
		XMR:   true,
		SOL:   true,
		ETH:   false,
		PAINT: false,
		HEX:   true,
		MATIC: false,
		BUSD:  true,
		SHIB:  false,
		PNK:   true,
	}

	user.CryptosEnabled = CryptosJsonStringToStruct(cryptosEnabled.String) // assign the sql.NullString to the user's "DonoGIF" field
	if !cryptosEnabled.Valid {                                             // check if the "dono_gif" column is null
		user.CryptosEnabled = ce // set the user's "DonoSound"
	}

	return user, nil
}

// check a user by their ID
func (ur *UserRepository) CheckUserByID(id int) bool {
	var user User
	var links, donoGIF, donoSound, defaultCrypto, alertURL, cryptosEnabled sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.Db.QueryRow("SELECT * FROM users WHERE id=?", id)
	err := row.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress,
		&user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono,
		&user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &links, &donoGIF, &donoSound, &alertURL, &user.DateEnabled, &user.WalletUploaded, &cryptosEnabled, &defaultCrypto)

	ce := CryptosEnabled{
		XMR:   true,
		SOL:   true,
		ETH:   false,
		PAINT: false,
		HEX:   true,
		MATIC: false,
		BUSD:  true,
		SHIB:  false,
		PNK:   true,
	}

	user.CryptosEnabled = CryptosJsonStringToStruct(cryptosEnabled.String) // assign the sql.NullString to the user's "DonoGIF" field
	if !cryptosEnabled.Valid {                                             // check if the "dono_gif" column is null
		user.CryptosEnabled = ce // set the user's "DonoSound"
	}

	user.DefaultCrypto = defaultCrypto.String // assign the sql.NullString to the user's "DonoGIF" field
	if !defaultCrypto.Valid {                 // check if the "dono_gif" column is null
		user.DefaultCrypto = "" // set the user's "DonoSound"
	}

	if err == sql.ErrNoRows {
		log.Println("checkUserByID(", id, "): User doesn't exist")
		return false // user doesn't exist
	} else if err != nil {
		log.Println("checkUserByID(", id, ") Error:", err)
		return false
	}
	return true // user exists

}

// check a user by their username and return a bool and the id
func (ur *UserRepository) CheckUserByUsername(username string) (bool, int) {
	ur.PrintUserColumns()
	var user User
	var links, donoGIF, donoSound, defaultCrypto, alertURL, cryptosEnabled sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.Db.QueryRow("SELECT * FROM users WHERE Username=?", username)
	err := row.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress,
		&user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono,
		&user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &links, &donoGIF, &donoSound, &alertURL, &user.DateEnabled, &user.WalletUploaded, &cryptosEnabled, &defaultCrypto)

	ce := CryptosEnabled{
		XMR:   true,
		SOL:   true,
		ETH:   false,
		PAINT: false,
		HEX:   true,
		MATIC: false,
		BUSD:  true,
		SHIB:  false,
		PNK:   true,
	}

	user.CryptosEnabled = CryptosJsonStringToStruct(cryptosEnabled.String) // assign the sql.NullString to the user's "DonoGIF" field
	if !cryptosEnabled.Valid {                                             // check if the "dono_gif" column is null
		user.CryptosEnabled = ce // set the user's "DonoSound"
	}

	user.DefaultCrypto = defaultCrypto.String // assign the sql.NullString to the user's "DonoGIF" field
	if !defaultCrypto.Valid {                 // check if the "dono_gif" column is null
		user.DefaultCrypto = "" // set the user's "DonoSound"
	}

	if err == sql.ErrNoRows {
		log.Println("checkUserByUsername(", username, "): User doesn't exist")
		return false, 0 // user doesn't exist
	} else if err != nil {
		log.Println("checkUserByUsername(", username, ") Error:", err)
		return false, 0
	}
	return true, user.UserID // user exists, return true and the user's ID
}

// get a user by their session token
func (ur *UserRepository) GetUserBySession(sessionToken string) (User, error) {
	userID, ok := ur.UserSessions[sessionToken]
	if !ok {
		return User{}, fmt.Errorf("session token not found")
	}
	var user User
	var links, donoGIF, defaultCrypto, donoSound, alertURL, cryptosEnabled sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.Db.QueryRow("SELECT * FROM users WHERE id=?", userID)
	err := row.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress,
		&user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono,
		&user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &links, &donoGIF, &donoSound, &alertURL, &user.DateEnabled, &user.WalletUploaded, &cryptosEnabled, &defaultCrypto)
	if err != nil {
		return User{}, err
	}
	user.Links = links.String
	if !links.Valid {
		user.Links = ""
	}
	user.DonoGIF = donoGIF.String // assign the sql.NullString to the user's "DonoGIF" field
	if !donoGIF.Valid {           // check if the "dono_gif" column is null
		user.DonoGIF = "default.gif" // set the user's "DonoGIF"
	}
	user.DonoSound = donoSound.String // assign the sql.NullString to the user's "DonoGIF" field
	if !donoSound.Valid {             // check if the "dono_gif" column is null
		user.DonoSound = "default.mp3" // set the user's "DonoSound"
	}
	user.DefaultCrypto = defaultCrypto.String // assign the sql.NullString to the user's "DonoGIF" field
	if !defaultCrypto.Valid {                 // check if the "dono_gif" column is null
		user.DefaultCrypto = "" // set the user's "DonoSound"
	}

	user.AlertURL = alertURL.String // assign the sql.NullString to the user's "DonoGIF" field
	if !alertURL.Valid {            // check if the "dono_gif" column is null
		user.AlertURL = GenerateUniqueURL() // set the user's "DonoSound"
	}

	ce := CryptosEnabled{
		XMR:   true,
		SOL:   true,
		ETH:   false,
		PAINT: false,
		HEX:   true,
		MATIC: false,
		BUSD:  true,
		SHIB:  false,
		PNK:   true,
	}

	user.CryptosEnabled = CryptosJsonStringToStruct(cryptosEnabled.String) // assign the sql.NullString to the user's "DonoGIF" field
	if !cryptosEnabled.Valid {                                             // check if the "dono_gif" column is null
		user.CryptosEnabled = ce // set the user's "DonoSound"
	}

	return user, nil
}

// get links for a user
func (ur *UserRepository) GetUserLinks(user User) ([]Link, error) {
	if user.Links == "" {
		// Insert default links for the user
		defaultLinks := []Link{
			{URL: "https://powerchat.live/paultown?tab=donation", Description: "Powerchat"},
			{URL: "https://cozy.tv/paultown", Description: "cozy.tv/paultown"},
			{URL: "http://twitter.paul.town/", Description: "Twitter"},
			{URL: "https://t.me/paultownreal", Description: "Telegram"},
			{URL: "http://notes.paul.town/", Description: "notes.paul.town"},
		}

		jsonLinks, err := json.Marshal(defaultLinks)
		if err != nil {
			return nil, err
		}

		user.Links = string(jsonLinks)
		if err := ur.Update(user); err != nil {
			return nil, err
		}

		return defaultLinks, nil
	}

	var links []Link
	if err := json.Unmarshal([]byte(user.Links), &links); err != nil {
		return nil, err
	}

	return links, nil
}

func (ur *UserRepository) GetUserCryptosEnabled(user User) (User, error) {

	user.CryptosEnabled.XMR = false
	user.CryptosEnabled.SOL = false
	user.CryptosEnabled.ETH = false
	user.CryptosEnabled.PAINT = true
	user.CryptosEnabled.HEX = false
	user.CryptosEnabled.MATIC = true
	user.CryptosEnabled.BUSD = false
	user.CryptosEnabled.SHIB = true
	user.CryptosEnabled.PNK = false

	return user, nil

}

func (ur *UserRepository) GetActiveXMRUsers() ([]*User, error) {
	var users []*User

	// Define the query to select the active XMR users
	query := `SELECT * FROM users WHERE wallet_uploaded = ?`

	// Execute the query
	rows, err := ur.Db.Query(query, true)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var user User
		err = rows.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress, &user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono, &user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &user.Links, &user.DonoGIF, &user.DonoSound, &user.AlertURL, &user.WalletUploaded, &user.DateEnabled)
		if err != nil {
			return nil, err
		}

		oneMonthAhead := user.DateEnabled.AddDate(0, 1, 0)
		if oneMonthAhead.After(time.Now().UTC()) {
			users = append(users, &user)
		}

	}
	return users, nil
}

func (ur *UserRepository) GetActiveETHUsers() ([]*User, error) {
	var users []*User

	// Define the query to select the active ETH users
	query := `SELECT * FROM users WHERE eth_address != ''`

	// Execute the query
	rows, err := ur.Db.Query(query)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var user User
		err = rows.Scan(&user.UserID, &user.Username, &user.HashedPassword, &user.EthAddress, &user.SolAddress, &user.HexcoinAddress, &user.XMRWalletPassword, &user.MinDono, &user.MinMediaDono, &user.MediaEnabled, &user.CreationDatetime, &user.ModificationDatetime, &user.Links, &user.DonoGIF, &user.DonoSound, &user.AlertURL, &user.WalletUploaded, &user.DateEnabled)
		if err != nil {
			return nil, err
		}

		oneMonthAhead := user.DateEnabled.AddDate(0, 1, 0)
		if oneMonthAhead.After(time.Now().UTC()) {
			users = append(users, &user)
		}
	}
	return users, nil
}

func (ur *UserRepository) UpdateEnabledDate(userID int) error {
	// Get the current time
	now := time.Now()

	// Update the user's enabled date in the database
	_, err := ur.Db.Exec("UPDATE users SET date_enabled=? WHERE id=?", now, userID)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepository) MapToCryptosEnabled(selectedCryptos map[string]bool) CryptosEnabled {
	// Create a new instance of the CryptosEnabled struct
	cryptosEnabled := CryptosEnabled{}

	// Set each field of the CryptosEnabled struct based on the corresponding value in the map
	cryptosEnabled.XMR = selectedCryptos["monero"]
	cryptosEnabled.SOL = selectedCryptos["solana"]
	cryptosEnabled.ETH = selectedCryptos["ethereum"]
	cryptosEnabled.PAINT = selectedCryptos["paint"]
	cryptosEnabled.HEX = selectedCryptos["hex"]
	cryptosEnabled.MATIC = selectedCryptos["matic"]
	cryptosEnabled.BUSD = selectedCryptos["busd"]
	cryptosEnabled.SHIB = selectedCryptos["shiba_inu"]
	cryptosEnabled.PNK = selectedCryptos["pnk"]

	// Return the populated CryptosEnabled struct
	return cryptosEnabled
}

func (ur *UserRepository) CreateSession(userID int) (string, error) {
	sessionToken := uuid.New().String()
	ur.UserSessions[sessionToken] = userID
	return sessionToken, nil
}

func (ur *UserRepository) ValidateSession(r *http.Request) (int, error) {
	sessionToken, err := r.Cookie("session_token")
	if err != nil {
		return 0, fmt.Errorf("no session token found")
	}
	userID, ok := ur.UserSessions[sessionToken.Value]
	if !ok {
		return 0, fmt.Errorf("invalid session token")
	}
	return userID, nil
}

func (ur *UserRepository) CheckWalletExists(userID int) bool {
	idstr := strconv.Itoa(userID)
	up := "users/" + idstr + "/monero/wallet"
	up_ := "users/" + idstr + "/monero/wallet.keys"

	if CheckFileExists(up) && CheckFileExists(up_) {
		return true
	} else {
		return false
	}
}
