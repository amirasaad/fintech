package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/amirasaad/fintech/infra"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/config"
	"github.com/amirasaad/fintech/pkg/service/account"
	"github.com/amirasaad/fintech/pkg/service/auth"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"golang.org/x/term"
)

var userID uuid.UUID

func main() {
	verbose := flag.Bool("v", false, "enable verbose output")
	flag.Parse()

	// Setup logging
	var logger *slog.Logger
	if !*verbose {
		log.SetOutput(io.Discard)
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	// Load application configuration
	cfg, err := config.LoadAppConfig(logger)
	if err != nil {
		_, _ = color.New(color.FgRed).Fprintln(os.Stderr, "Failed to load application configuration:", err)
		return
	}

	// Log configuration details if verbose
	if *verbose {
		logger.Info("Configuration loaded successfully",
			"database_url", cfg.DB.Url,
			"jwt_expiry", cfg.Auth,
			"exchange_rate_api_configured", cfg.Exchange.ApiKey != "")
	}

	appEnv := os.Getenv("APP_ENV")
	// Initialize DB connection ONCE
	db, err := infra.NewDBConnection(cfg.DB, appEnv)
	if err != nil {
		_, _ = color.New(color.FgRed).Fprintln(os.Stderr, "Failed to initialize database:", err)
		return
	}

	// Create UOW factory using the shared db
	uow := infra_repository.NewUoW(db)

	// Create exchange rate system
	currencyConverter, err := infra.NewExchangeRateSystem(logger, *cfg)
	if err != nil {
		_, _ = color.New(color.FgRed).Fprintln(os.Stderr, "Failed to initialize exchange rate system:", err)
		return
	}

	scv := account.NewAccountService(uow, currencyConverter, logger)
	authSvc := auth.NewBasicAuthService(uow, logger)

	cliApp(scv, authSvc)
}

func cliApp(scv *account.AccountService, authSvc *auth.AuthService) {
	reader := bufio.NewReader(os.Stdin)
	banner := color.New(color.FgCyan, color.Bold).SprintFunc()
	prompt := color.New(color.FgGreen, color.Bold).SprintFunc()
	errorMsg := color.New(color.FgRed, color.Bold).SprintFunc()
	successMsg := color.New(color.FgHiBlue, color.Bold).SprintFunc()
	fmt.Println(banner(`
	███████╗██╗███╗   ██╗████████╗███████╗ ██████╗██╗  ██╗     ██████╗██╗     ██╗
	██╔════╝██║████╗  ██║╚══██╔══╝██╔════╝██╔════╝██║  ██║    ██╔════╝██║     ██║
	█████╗  ██║██╔██╗ ██║   ██║   █████╗  ██║     ███████║    ██║     ██║     ██║
	██╔══╝  ██║██║╚██╗██║   ██║   ██╔══╝  ██║     ██╔══██║    ██║     ██║     ██║
	██║     ██║██║ ╚████║   ██║   ███████╗╚██████╗██║  ██║    ╚██████╗███████╗██║
	╚═╝     ╚═╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝ ╚═════╝╚═╝  ╚═╝     ╚═════╝╚══════╝╚═╝
        								Version (v1.0.0)
	`))
	for {
		if userID == uuid.Nil {
			fmt.Println(prompt("Please login to continue."))
			fmt.Print(prompt("Username or Email: "))
			identity, _ := reader.ReadString('\n')
			identity = strings.TrimSpace(identity)
			fmt.Print(prompt("Password: "))
			bytePassword, _ := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			password := string(bytePassword)
			user, err := authSvc.Login(context.Background(), identity, password)
			if err != nil {
				fmt.Println(errorMsg("Login error:"), err)
				continue
			}
			if user == nil {
				fmt.Println(errorMsg("Invalid credentials."))
				continue
			}
			userID = user.ID
			fmt.Println(successMsg("Login successful!"))
		}
		fmt.Println(banner("\nAvailable commands: create, deposit <account_id> <amount>, withdraw <account_id> <amount>, balance <account_id>, logout, exit"))
		fmt.Print(prompt("> "))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "exit" {
			fmt.Println(successMsg("Goodbye!"))
			return
		}
		if input == "logout" {
			userID = uuid.Nil
			fmt.Println(successMsg("Logged out."))
			continue
		}
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}
		cmd := args[0]
		switch cmd {
		case "create":
			account, err := scv.CreateAccount(context.Background(), userID)
			if err != nil {
				fmt.Println(errorMsg("Error creating account:"), err)
				continue
			}
			balance, err := scv.GetBalance(userID, account.ID)
			if err != nil {
				fmt.Println(errorMsg("Error fetching account balance:"), err)
				continue
			}
			fmt.Println(successMsg(fmt.Sprintf("Account created: ID=%s, Balance=%.2f", account.ID, balance)))
		case "deposit":
			if len(args) < 3 {
				fmt.Println(errorMsg("Usage: deposit <account_id> <amount>"))
				continue
			}
			accountID := args[1]
			amount, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				fmt.Println(errorMsg("Invalid amount:"), err)
				continue
			}
			account, _, err := scv.Deposit(userID, uuid.MustParse(accountID), amount, "USD")
			if err != nil {
				fmt.Println(errorMsg("Error depositing:"), err)
				continue
			}
			balance, err := scv.GetBalance(userID, uuid.MustParse(accountID))
			if err != nil {
				fmt.Println(errorMsg("Error fetching account balance:"), err)
				continue
			}
			fmt.Println(successMsg(fmt.Sprintf("Deposited %.2f to account %s. New balance: %.2f", amount, account.ID, balance)))
		case "withdraw":
			if len(args) < 3 {
				fmt.Println(errorMsg("Usage: withdraw <account_id> <amount>"))
				continue
			}
			accountID := args[1]
			amount, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				fmt.Println(errorMsg("Invalid amount:"), err)
				continue
			}
			account, _, err := scv.Withdraw(userID, uuid.MustParse(accountID), amount, "USD")
			if err != nil {
				fmt.Println(errorMsg("Error withdrawing:"), err)
				continue
			}
			balance, err := scv.GetBalance(userID, account.ID)
			if err != nil {
				fmt.Println(errorMsg("Error fetching account balance:"), err)
				continue
			}
			fmt.Println(successMsg(fmt.Sprintf("Withdrew %.2f from account %s. New balance: %.2f", amount, account.ID, balance)))
		case "balance":
			if len(args) < 2 {
				fmt.Println(errorMsg("Usage: balance <account_id>"))
				continue
			}
			accountID := args[1]
			balance, err := scv.GetBalance(userID, uuid.MustParse(accountID))
			if err != nil {
				fmt.Println(errorMsg("Error fetching balance:"), err)
				continue
			}
			fmt.Println(successMsg(fmt.Sprintf("Account %s balance: %.2f", accountID, balance)))
		default:
			fmt.Println(errorMsg("Unknown command:"), cmd)
		}
	}
}
