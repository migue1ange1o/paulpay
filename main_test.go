package main

import (
	"database/sql"
	"log"
	"shadowchat/internal/models"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	var a models.AlertPageData
	var pb models.ProgressbarData

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
	dr := models.NewDonoRepository(db, ur, mr)

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

	// Schedule a function to run fetchExchangeRates every three minutes
	go models.FetchExchangeRates(ur)
	go dr.Check()
	go models.CheckPendingAccounts(dr)
	go models.CheckBillingAccounts(dr)

	go checkAccountBillings()

	a.Refresh = 10
	pb.Refresh = 1
	_ = ur.GetObsData(1)
	inviteCodeMap = getAllCodes()
	models.SetServerVars(ur)

	go dr.CreateTestDono(2, "Big Bob", "XMR", "This Cruel Message is Bob's Test message! Test message! Test message! Test message! Test message! Test message! Test message! Test message! Test message! Test message! ", "50", 100, "https://www.youtube.com/watch?v=6iseNlvH2_s")
	go dr.CreateTestDono(3, "Medium Bob", "XMR", "Hey it's medium Bob ", "0.1", 3, "https://www.youtube.com/watch?v=6iseNlvH2_s")

	log.Println("Check for accounts complete")

}
