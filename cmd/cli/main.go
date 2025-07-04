package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/pkg/repository"
	"github.com/amirasaad/fintech/pkg/service"
	"github.com/google/uuid"
	"golang.org/x/term"
)

var userID uuid.UUID

func main() {
	db, err := infra.NewDBConnection()
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}
	uowFactory := func() (repository.UnitOfWork, error) {
		return infra.NewGormUoW(db)
	}
	scv := service.NewAccountService(uowFactory)
	authSvc := service.NewBasicAuthService(uowFactory)

	cliApp(scv, authSvc)
}

func cliApp(scv *service.AccountService, authSvc *service.AuthService) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to the Fintech CLI!")
	for {
		if userID == uuid.Nil {
			fmt.Println("Please login to continue.")
			fmt.Print("Username or Email: ")
			identity, _ := reader.ReadString('\n')
			identity = strings.TrimSpace(identity)
			fmt.Print("Password: ")
			bytePassword, _ := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			password := string(bytePassword)
			user, _, err := authSvc.Login(identity, password)
			if err != nil {
				fmt.Println("Login error:", err)
				continue
			}
			if user == nil {
				fmt.Println("Invalid credentials.")
				continue
			}
			userID = user.ID
			fmt.Println("Login successful!")
		}
		fmt.Println("\nAvailable commands: create, deposit <account_id> <amount>, withdraw <account_id> <amount>, balance <account_id>, logout, exit")
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "exit" {
			fmt.Println("Goodbye!")
			return
		}
		if input == "logout" {
			userID = uuid.Nil
			fmt.Println("Logged out.")
			continue
		}
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}
		cmd := args[0]
		switch cmd {
		case "create":
			account, err := scv.CreateAccount(userID)
			if err != nil {
				fmt.Println("Error creating account:", err)
				continue
			}
			fmt.Printf("Account created: ID=%s, Balance=%d\n", account.ID, account.Balance)
		case "deposit":
			if len(args) < 3 {
				fmt.Println("Usage: deposit <account_id> <amount>")
				continue
			}
			accountID := args[1]
			amount, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				fmt.Println("Invalid amount:", err)
				continue
			}
			account, err := scv.Deposit(userID, uuid.MustParse(accountID), amount)
			if err != nil {
				fmt.Println("Error depositing:", err)
				continue
			}
			fmt.Printf("Deposited %.2f to account %s. New balance: %.d\n", amount, account.ID, account.Balance)
		case "withdraw":
			if len(args) < 3 {
				fmt.Println("Usage: withdraw <account_id> <amount>")
				continue
			}
			accountID := args[1]
			amount, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				fmt.Println("Invalid amount:", err)
				continue
			}
			account, err := scv.Withdraw(userID, uuid.MustParse(accountID), amount)
			if err != nil {
				fmt.Println("Error withdrawing:", err)
				continue
			}
			fmt.Printf("Withdrew %.2f from account %s. New balance: %d\n", amount, account.ID, account.Balance)
		case "balance":
			if len(args) < 2 {
				fmt.Println("Usage: balance <account_id>")
				continue
			}
			accountID := args[1]
			account, err := scv.GetAccount(userID, uuid.MustParse(accountID))
			if err != nil {
				fmt.Println("Error fetching balance:", err)
				continue
			}
			fmt.Printf("Account %s balance: %d\n", account.ID, account.Balance)
		default:
			fmt.Println("Unknown command:", cmd)
		}
	}
}
