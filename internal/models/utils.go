package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"shadowchat/utils"
	"time"
)

var starting_port int = 28088

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

	err = createObsTable(db)
	if err != nil {
		log.Fatal(err)
	}

	ur.createAdmin()
	ur.createNew("paul", "hunter")

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
		value := utils.GenerateUniqueURL()
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

func cryptosStructToJSONString(s CryptosEnabled) string {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Println("cryptosStructToJSONString error:", err)
		return ""
	}
	return string(bytes)
}

func cryptosJsonStringToStruct(jsonStr string) CryptosEnabled {
	var s CryptosEnabled
	err := json.Unmarshal([]byte(jsonStr), &s)
	if err != nil {
		log.Println("cryptosJsonStringToStruct error:", err)
		return CryptosEnabled{}
	}
	return s
}

func createObsTable(db *sql.DB) error {
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
	ur.printUserColumns()
	users, err := ur.getAll()
	if err != nil {
		log.Fatalf("startWallet() error: %v", err)
	}

	for _, user := range users {
		log.Println("Checking user:", user.Username, "User ID:", user.UserID, "User billing data enabled:", user.BillingData.Enabled)
		if user.BillingData.Enabled {
			log.Println("User valid", user.UserID, "User eth_address:", globalUsers[user.UserID].EthAddress)
			if user.WalletUploaded {
				log.Println("Monero wallet uploaded")
				mr.xmrWallets = append(mr.xmrWallets, []int{user.UserID, starting_port})
				go mr.startMoneroWallet(starting_port, user.UserID, user)
				starting_port++

			} else {
				log.Println("Monero wallet not uploaded")
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
