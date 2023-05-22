package models

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"shadowchat/utils"

	//	"github.com/davecgh/go-spew/spew"

	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var globalUsers = map[int]utils.User{}
var pendingGlobalUsers = map[int]utils.PendingUser{}

var db *sql.DB
var userSessions = make(map[string]int)
var amountNeeded = 1000.00
var amountSent = 200.00
var donosMap = make(map[int]utils.Dono) // initialize an empty map
var minDonoValue float64 = 5.0

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

type UserRepository struct {
	db          *sql.DB
	solRepo     *SolRepository
	billingRepo *BillingRepository
	users       map[int]User
}

func NewUserRepository(db *sql.DB, sr *SolRepository, br *BillingRepository) *UserRepository {
	return &UserRepository{
		db:          db,
		solRepo:     sr,
		billingRepo: br,
		users:       make(map[int]User),
	}
}

func (ur *UserRepository) getByID(userID int) (User, error) {
	user, ok := ur.users[userID]
	if !ok {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func (ur *UserRepository) createAdmin() {

	ur.createNew("admin", "hunter123")
}

func (ur *UserRepository) getNew(username string, hashedPassword []byte) User {

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
		AlertURL:          utils.GenerateUniqueURL(),
		WalletUploaded:    false,
		Links:             "",
		DateEnabled:       time.Now().UTC(),
	}
	return user
}

func (ur *UserRepository) createNew(username, password string) error {
	log.Println("running createNewUser")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		return err
	}
	// create admin user if not exists
	user := ur.getNew(username, hashedPassword)
	userID := ur.create(user)
	if userID != 0 {
		ur.createNewOBS(userID, "default message", 100.00, 50.00, 5, user.DonoGIF, user.DonoSound, "test_voice")
		log.Println("createUser() succeeded, so OBS row was created.")
	} else {
		log.Println("createUser() didn't succeed, so OBS row wasn't created.")
	}

	log.Println("finished createNewUser")
	return nil
}

func (ur *UserRepository) create(user User) int {
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

	ce_ := cryptosStructToJSONString(ce)

	_, err := ur.db.Exec(`
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
	row := ur.db.QueryRow(`SELECT last_insert_rowid()`)
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

	_, err = ur.db.Exec(`
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
	ur.users[userID] = user

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

	_, err = ur.getAll()
	if err != nil {
		log.Fatalf("createUser() getAllUsers() error: %v", err)
	}

	return userID
}

func (ur *UserRepository) getAll() ([]User, error) {
	var users []User
	rows, err := ur.db.Query("SELECT * FROM users")
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

		user.CryptosEnabled = cryptosJsonStringToStruct(cryptosEnabled.String)
		if !cryptosEnabled.Valid {
			log.Println("user cryptos enabled not fixed")
			user.CryptosEnabled = ce
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	billings, err := ur.billingRepo.getAllBilling()
	if err != nil {
		return nil, err
	}

	for _, billing := range billings {
		ur.billingRepo.billings[billing.UserID] = billing
	}

	for i := range users {
		billing, ok := ur.billingRepo.billings[users[i].UserID]
		if ok {
			users[i].BillingData = billing
			ur.users[users[i].UserID] = users[i]
		}
	}

	return users, nil
}

// old: updateUser
func (ur *UserRepository) update(user User) error {
	ur.users[user.UserID] = user
	statement := `
		UPDATE users
		SET Username=?, HashedPassword=?, eth_address=?, sol_address=?, hex_address=?,
			xmr_wallet_password=?, min_donation_threshold=?, min_media_threshold=?, media_enabled=?, modified_at=?, links=?, dono_gif=?, dono_sound=?, alert_url=?, date_enabled=?, wallet_uploaded=?, cryptos_enabled=?
		WHERE id=?
	`
	_, err := ur.db.Exec(statement, user.Username, user.HashedPassword, user.EthAddress,
		user.SolAddress, user.HexcoinAddress, user.XMRWalletPassword, user.MinDono, user.MinMediaDono,
		user.MediaEnabled, time.Now().UTC(), user.Links, user.DonoGIF, user.DonoSound, user.AlertURL, user.DateEnabled, user.WalletUploaded, cryptosStructToJSONString(user.CryptosEnabled), user.UserID)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}

	statement = `
		UPDATE billing
		SET user_id=?, amount_this_month=?, amount_total=?, enabled=?, need_to_pay=?,
			eth_amount=?, xmr_amount=?, xmr_pay_id=?, created_at=?, updated_at=?
		WHERE billing_id=?
	`
	_, err = ur.db.Exec(statement, user.UserID, user.BillingData.AmountThisMonth, user.BillingData.AmountTotal, user.BillingData.Enabled,
		user.BillingData.NeedToPay, user.BillingData.ETHAmount, user.BillingData.XMRAmount, user.BillingData.XMRPayID, user.BillingData.CreatedAt,
		user.BillingData.UpdatedAt, user.BillingData.UserID)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}

	ur.solRepo.wallets[user.UserID] = SolWallet{
		Address: user.SolAddress,
		Amount:  0.00,
	}

	ur.solRepo.SetSolWallets(ur.solRepo.wallets)
	return err
}

func (ur *UserRepository) updateObsData(userID int, gifName string, mp3Name string, ttsVoice string, pbData ProgressbarData) error {
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
	_, err := ur.db.Exec(updateObsData, userID, gifName, mp3Name, ttsVoice, pbData.Message, pbData.Needed, pbData.Sent, userID)
	return err
}

func (ur *UserRepository) createNewOBS(userID int, message string, needed, sent float64, refresh int, gifFile, soundFile, ttsVoice string) {
	pbData := utils.ProgressbarData{
		Message: message,
		Needed:  needed,
		Sent:    sent,
		Refresh: refresh,
	}
	err := ur.insertObsData(userID, gifFile, soundFile, ttsVoice, pbData)
	if err != nil {
		log.Fatal(err)
	}

}

func (ur *UserRepository) insertObsData(userId int, gifName, mp3Name, ttsVoice string, pbData utils.ProgressbarData) error {
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
	_, err := ur.db.Exec(obsData, userId, gifName, mp3Name, ttsVoice, pbData.Message, pbData.Needed, pbData.Sent)
	return err
}

func (ur *UserRepository) getOBSDataByUserID(userID int) (utils.OBSDataStruct, error) {
	var obsData utils.OBSDataStruct
	//var alertURL sql.NullString // use sql.NullString for the "links" and "dono_gif" fields
	row := ur.db.QueryRow("SELECT gif_name, mp3_name, `message`, needed, sent FROM obs WHERE user_id=?", userID)

	err := row.Scan(&obsData.FilenameGIF, &obsData.FilenameMP3, &obsData.Message, &obsData.Needed, &obsData.Sent)
	if err != nil {
		log.Println("Couldn't get obsData,", err)
		return obsData, err
	}

	return obsData, nil
}

func (ur *UserRepository) printUserColumns() error {
	rows, err := ur.db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'users';`)
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
