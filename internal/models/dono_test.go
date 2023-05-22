// models_test/dono_test.go
package models

import (
	"database/sql"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDonoRepository_CheckUnfulfilled(t *testing.T) {
	// Create a new instance of DonoRepository
	// Open a new database connection
	var err error
	db, err = sql.Open("sqlite3", "users.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sr := NewSolRepository(db)
	br := NewBillingRepository(db)
	ur := NewUserRepository(db, sr, br)
	mr := NewMoneroRepository(db, ur)
	dr := NewDonoRepository(db, ur, mr)

	// Check if the database and tables exist, and create them if they don't
	err = CreateDatabaseIfNotExists(db, ur)
	if err != nil {
		log.Printf("Error creating database: %v", err)
		panic(err)
	}

	// Run migrations on database
	err = RunDatabaseMigrations(db)
	if err != nil {
		log.Printf("Error migrating database: %v", err)
		panic(err)
	}

	go StartWallets(ur, mr, sr)

	time.Sleep(5 * time.Second)
	log.Println("Starting server")

	// Generate some test donations
	donations := []Dono{
		{
			ID:           1,
			UserID:       123,
			Address:      "0xF9faa1851f55536dbfAf2dF8137191F85CcE3f56",
			Name:         "miguel",
			Message:      "hello",
			AmountToSend: "0.00481551",
			AmountSent:   "0.00481551",
			CurrencyType: "ETH",
			AnonDono:     false,
			Fulfilled:    false,
			EncryptedIP:  "ABC123",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			USDAmount:    10.0,
			MediaURL:     "https://example.com/media",
		},
		{
			ID:           2,
			UserID:       123,
			Address:      "0xF9faa1851f55536dbfAf2dF8137191F85CcE3f56",
			Name:         "miguel",
			Message:      "hello",
			AmountToSend: "0.00481551",
			AmountSent:   "0.00481551",
			CurrencyType: "ETH",
			AnonDono:     false,
			Fulfilled:    false,
			EncryptedIP:  "DEF456",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			USDAmount:    25.0,
			MediaURL:     "https://example.com/another-media",
		},
	}

	// Add test data to DonoRepository
	for _, dono := range donations {
		dr.donos[dono.ID] = dono
	}

	// Call the CheckUnfulfilled method
	fulfilledDonos, err := dr.checkUnfulfilled()
	if err != nil {
		log.Printf("complete failure: %s", err)
	}
	// Assert the expected behavior
	assert.NoError(t, err)
	assert.Equal(t, len(dr.donos), len(fulfilledDonos)) // All unfulfilled donations should be marked as fulfilled

	// Verify the data integrity
	for _, dono := range fulfilledDonos {
		assert.True(t, dono.Fulfilled)      // Donations should be marked as fulfilled
		assert.NotEmpty(t, dono.AmountSent) // AmountSent should not be empty
	}
}
