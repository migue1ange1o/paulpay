package models

import (
	"database/sql"
	"log"
	"math/rand"
	"time"
)

type InviteCode struct {
	Value  string
	Active bool
}

type InviteRepositoryInterface interface {
	GetAllCodes() map[string]InviteCode
	CreateNewInviteCode(value string, active bool) error
	UpdateInviteCode(code InviteCode) error
	GenerateMoreInviteCodes(codeAmount int)
	CheckValidInviteCode(ic string) bool
	GenerateUniqueCode() string
	GenerateUniqueCodes(amount int) map[string]InviteCode
	AddInviteCodes(existingMap map[string]InviteCode, newMap map[string]InviteCode) map[string]InviteCode
}

type InviteRepository struct {
	Db            *sql.DB
	InviteCodeMap map[string]InviteCode
}

func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{
		Db:            db,
		InviteCodeMap: map[string]InviteCode{},
	}
}

func (ir *InviteRepository) GetAllCodes() map[string]InviteCode {
	rows, err := ir.Db.Query("SELECT * FROM invites")
	if err != nil {
		log.Println(err)
		return ir.InviteCodeMap
	}
	defer rows.Close()

	for rows.Next() {
		var ic InviteCode

		err = rows.Scan(&ic.Value, &ic.Active)

		if err != nil {
			log.Println(err)
			return ir.InviteCodeMap
		}
		ir.InviteCodeMap[ic.Value] = ic
	}

	return ir.InviteCodeMap
}

func (ir *InviteRepository) CreateNewInviteCode(value string, active bool) error {
	inviteData := `
        INSERT INTO invites (
            value,
            active
        ) VALUES (?, ?);`
	_, err := ir.Db.Exec(inviteData, value, active)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (ir *InviteRepository) UpdateInviteCode(code InviteCode) error {
	ir.InviteCodeMap[code.Value] = code
	statement := `
		UPDATE invites
		SET value=?, active=?
		WHERE value=?
	`
	_, err := ir.Db.Exec(statement, code.Value, code.Active, code.Value)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}
	return err
}

func (ir *InviteRepository) GenerateMoreInviteCodes(codeAmount int) {
	newCodes := ir.GenerateUniqueCodes(codeAmount)
	for _, code := range newCodes {
		err := ir.CreateNewInviteCode(code.Value, code.Active)
		if err != nil {
			log.Println("createNewInviteCode() error:", err)
		}
	}
	ir.InviteCodeMap = ir.AddInviteCodes(ir.InviteCodeMap, newCodes)
}

func (ir *InviteRepository) CheckValidInviteCode(ic string) bool {
	if _, ok := ir.InviteCodeMap[ic]; ok {
		if ir.InviteCodeMap[ic].Active {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (ir *InviteRepository) GenerateUniqueCode() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const length = 15
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	return (string(randomString))
}

func (ir *InviteRepository) GenerateUniqueCodes(amount int) map[string]InviteCode {
	inviteCodes := make(map[string]InviteCode)
	for i := 0; i < amount; i++ {
		cS := ir.GenerateUniqueCode()
		inviteCodes[cS] = InviteCode{Value: cS, Active: true}
	}

	return inviteCodes
}

func (ir *InviteRepository) AddInviteCodes(existingMap map[string]InviteCode, newMap map[string]InviteCode) map[string]InviteCode {
	for key, value := range newMap {
		existingMap[key] = value
	}

	return existingMap
}
