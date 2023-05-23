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
	getAllCodes() map[string]InviteCode
	createNewInviteCode(value string, active bool) error
	updateInviteCode(code InviteCode) error
	generateMoreInviteCodes(codeAmount int)
	checkValidInviteCode(ic string) bool
}

type InviteRepository struct {
	db            *sql.DB
	inviteCodeMap map[string]InviteCode
}

func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{
		db:            db,
		inviteCodeMap: map[string]InviteCode{},
	}
}

func (ir *InviteRepository) getAllCodes() map[string]InviteCode {
	rows, err := ir.db.Query("SELECT * FROM invites")
	if err != nil {
		log.Println(err)
		return ir.inviteCodeMap
	}
	defer rows.Close()

	for rows.Next() {
		var ic InviteCode

		err = rows.Scan(&ic.Value, &ic.Active)

		if err != nil {
			log.Println(err)
			return ir.inviteCodeMap
		}
		ir.inviteCodeMap[ic.Value] = ic
	}

	return ir.inviteCodeMap
}

func (ir *InviteRepository) createNewInviteCode(value string, active bool) error {
	inviteData := `
        INSERT INTO invites (
            value,
            active
        ) VALUES (?, ?);`
	_, err := ir.db.Exec(inviteData, value, active)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (ir *InviteRepository) updateInviteCode(code InviteCode) error {
	ir.inviteCodeMap[code.Value] = code
	statement := `
		UPDATE invites
		SET value=?, active=?
		WHERE value=?
	`
	_, err := ir.db.Exec(statement, code.Value, code.Active, code.Value)
	if err != nil {
		log.Fatalf("failed, err: %v", err)
	}
	return err
}

func (ir *InviteRepository) generateMoreInviteCodes(codeAmount int) {
	newCodes := ir.GenerateUniqueCodes(codeAmount)
	for _, code := range newCodes {
		err := ir.createNewInviteCode(code.Value, code.Active)
		if err != nil {
			log.Println("createNewInviteCode() error:", err)
		}
	}
	ir.inviteCodeMap = ir.AddInviteCodes(ir.inviteCodeMap, newCodes)
}

func (ir *InviteRepository) checkValidInviteCode(ic string) bool {
	if _, ok := ir.inviteCodeMap[ic]; ok {
		if ir.inviteCodeMap[ic].Active {
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
