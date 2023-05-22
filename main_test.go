package main

import (
	"database/sql"
	"log"
	"shadowchat/internal/models"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	// Create a new instance of DonoRepository
	// Open a new database connection
	var err error
	db, err = sql.Open("sqlite3", "users.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sr := models.NewSolRepository(db)
	br := models.NewBillingRepository(db)
	ur := models.NewUserRepository(db, sr, br)
	mr := models.NewMoneroRepository(db, ur)
	// dr := models.NewDonoRepository(db, ur, mr)

	// Check if the database and tables exist, and create them if they don't
	err = models.CreateDatabaseIfNotExists(db, ur)
	if err != nil {
		log.Printf("Error creating database: %v", err)
	}

	// Run migrations on database
	err = models.RunDatabaseMigrations(db)
	if err != nil {
		log.Printf("Error migrating database: %v", err)
	}

	go models.StartWallets(ur, mr, sr)

	time.Sleep(5 * time.Second)
	log.Println("Starting server")
}
