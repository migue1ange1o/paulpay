package models

import (
	"database/sql"
	"fmt"
	"log"

	//"github.com/davecgh/go-spew/spew"
	//bin "github.com/gagliardetto/binary"

	"time"
)

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
type BillingRepositoryInterface interface {
	getAllBilling() ([]BillingData, error)
}

type BillingRepository struct {
	db       *sql.DB
	billings map[int]BillingData
}

func NewBillingRepository(db *sql.DB) *BillingRepository {
	return &BillingRepository{
		db:       db,
		billings: map[int]BillingData{},
	}
}
func (br *BillingRepository) getAllBilling() ([]BillingData, error) {
	var billings []BillingData
	rows, err := br.db.Query("SELECT * FROM billing")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var billingData BillingData
		err = rows.Scan(&billingData.BillingID, &billingData.UserID, &billingData.AmountThisMonth, &billingData.AmountTotal, &billingData.Enabled, &billingData.NeedToPay, &billingData.ETHAmount, &billingData.XMRAmount, &billingData.XMRPayID, &billingData.CreatedAt, &billingData.UpdatedAt)
		if err != nil {
			log.Println(err)
		}

		fmt.Println("UserID: ", billingData.UserID)
		fmt.Println("Amount This Month: ", billingData.AmountThisMonth)
		fmt.Println("Amount Total: ", billingData.AmountTotal)
		fmt.Println("Enabled: ", billingData.Enabled)
		fmt.Println("Need To Pay: ", billingData.NeedToPay)
		fmt.Println("Created At: ", billingData.CreatedAt)
		fmt.Println("Updated At: ", billingData.UpdatedAt)

		billings = append(billings, billingData)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return billings, nil
}
