package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"shadowchat/utils"

	//	"github.com/davecgh/go-spew/spew"

	"time"

	_ "github.com/mattn/go-sqlite3"
)

var globalUsers = map[int]utils.User{}
var pendingGlobalUsers = map[int]utils.PendingUser{}

var db *sql.DB
var userSessions = make(map[string]int)
var amountNeeded = 1000.00
var amountSent = 200.00
var donosMap = make(map[int]utils.Dono) // initialize an empty map

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
	CryptosEnabled       CryptosEnabled
	BillingData          BillingData
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

type BillingData struct {
	BillingID       int
	UserID          int
	AmountThisMonth float64
	AmountTotal     float64
	AmountNeeded    float64
	ETHAmount       string
	XMRAmount       string
	XMRPayID        string
	XMRAddress      string
	Enabled         bool
	NeedToPay       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type UserRepository struct {
	db      *sql.DB
	solRepo *SolRepository
	users   map[int]*User
}

func NewUserRepository(db *sql.DB, sr *SolRepository) *UserRepository {
	return &UserRepository{
		db:      db,
		solRepo: sr,
		users:   make(map[int]*User),
	}
}

func (ur *UserRepository) getByID(userID int) (*User, error) {
	user, ok := ur.users[userID]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// old: updateUser
func (ur *UserRepository) update(user *User) error {
	ur.users[user.UserID] = user
	statement := `
		UPDATE users
		SET Username=?, HashedPassword=?, eth_address=?, sol_address=?, hex_address=?,
			xmr_wallet_password=?, min_donation_threshold=?, min_media_threshold=?, media_enabled=?, modified_at=?, links=?, dono_gif=?, dono_sound=?, alert_url=?, date_enabled=?, wallet_uploaded=?, cryptos_enabled=?
		WHERE id=?
	`
	_, err := db.Exec(statement, user.Username, user.HashedPassword, user.EthAddress,
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
	_, err = db.Exec(statement, user.UserID, user.BillingData.AmountThisMonth, user.BillingData.AmountTotal, user.BillingData.Enabled,
		user.BillingData.NeedToPay, user.BillingData.ETHAmount, user.BillingData.XMRAmount, user.BillingData.XMRPayID, user.BillingData.CreatedAt,
		user.BillingData.UpdatedAt, user.BillingData.UserID)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}

	ur.solRepo.wallets[user.UserID] = SolWallet{
		Address: user.SolAddress,
		Amount:  0.00,
	}

	SetSolWallets(ur.solRepo.wallets)
	return err
}

func cryptosStructToJSONString(s CryptosEnabled) string {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Println("cryptosStructToJSONString error:", err)
		return ""
	}
	return string(bytes)
}
