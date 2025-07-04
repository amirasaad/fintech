package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
)

func main() {
	argsLen := len(os.Args)
	if argsLen < 2 {
		fmt.Println("Usage: cli <command> [arguments]")
		fmt.Println("Commands: create, deposit <account_id> <amount>, withdraw <account_id> <amount>, balance <account_id>")
		return
	}
	cmd := os.Args[1]
	db, err := infra.NewDBConnection()
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	scv := service.NewAccountService(func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	})
	userID := uuid.New()
	switch cmd {
	case "create":
		account, err := scv.CreateAccount(userID)
		fmt.Println(account)
		if err != nil {
			fmt.Println("Error creating account:", err)
			return
		}
		balance, err := scv.GetBalance(userID, account.ID)
		if err != nil {
			fmt.Println("Error fetching balance:", err)
			return
		}
		fmt.Printf("Account created: ID=%s, Balance=%.2f\n", account.ID, balance)
	case "deposit":
		if argsLen < 4 {
			fmt.Println("Usage: deposit <account_id> <amount>")
			return
		}
		accountID := os.Args[2]
		amount, err := strconv.ParseFloat(os.Args[3], 64)
		if err != nil {
			fmt.Println("Invalid amount:", err)
			return
		}
		account, err := scv.Deposit(userID, uuid.MustParse(accountID), amount)
		if err != nil {
			fmt.Println("Error depositing:", err)
			return
		}
		balance, err := scv.GetBalance(userID, account.ID)
		if err != nil {
			fmt.Println("Error fetching balance:", err)
			return
		}
		fmt.Printf("Deposited %.2f to account %s. New balance: %.2f\n", amount, account.ID, balance)
	case "withdraw":
		if argsLen < 4 {
			fmt.Println("Usage: withdraw <account_id> <amount>")
			return
		}
		accountID := os.Args[2]
		amount, err := strconv.ParseFloat(os.Args[3], 64)
		if err != nil {
			fmt.Println("Invalid amount:", err)
			return
		}
		account, err := scv.Withdraw(userID, uuid.MustParse(accountID), amount)
		if err != nil {
			fmt.Println("Error withdrawing:", err)
			return
		}
		balance, err := scv.GetBalance(userID, account.ID)
		if err != nil {
			fmt.Println("Error fetching balance:", err)
			return
		}
		fmt.Printf("Withdrew %.2f from account %s. New balance: %.2f\n", amount, account.ID, balance)
	case "balance":
		if argsLen < 3 {
			fmt.Println("Usage: balance <account_id>")
			return
		}
		accountID := os.Args[2]
		account, err := scv.GetAccount(userID, uuid.MustParse(accountID))
		if err != nil {
			fmt.Println("Error fetching balance:", err)
			return
		}
		balance, err := scv.GetBalance(userID, account.ID)
		if err != nil {
			fmt.Println("Error fetching balance:", err)
			return
		}
		fmt.Printf("Account %s balance: %.2f\n", account.ID, balance)
	default:
		fmt.Println("Unknown command:", cmd)
	}
}
