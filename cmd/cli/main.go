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

	"github.com/amirasaad/fintech/config"
	"github.com/amirasaad/fintech/infra"
	"github.com/amirasaad/fintech/infra/provider"
	infra_repository "github.com/amirasaad/fintech/infra/repository"
	"github.com/amirasaad/fintech/pkg/commands"
	"github.com/amirasaad/fintech/pkg/dto"
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

	scv := account.NewService(config.Deps{
		Uow:               uow,
		CurrencyConverter: currencyConverter,
		Logger:            logger,
		PaymentProvider:   provider.NewMockPaymentProvider(),
	})
	authSvc := auth.NewBasicAuthService(uow, logger)

	cliApp(scv, authSvc)
}

func cliApp(scv *account.Service, authSvc *auth.AuthService) {
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
		if !handleLogin(reader, prompt, errorMsg, successMsg, authSvc) {
			continue
		}

		fmt.Println(banner("\nAvailable commands: create, deposit <account_id> <amount>, withdraw <account_id> <amount>, balance <account_id>, logout, exit"))
		fmt.Print(prompt("> "))

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if handleSpecialCommands(input, successMsg) {
			continue
		}

		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		handleCommand(args, scv, errorMsg, successMsg)
	}
}

func handleLogin(reader *bufio.Reader, prompt, errorMsg, successMsg func(a ...interface{}) string, authSvc *auth.AuthService) bool {
	if userID != uuid.Nil {
		return true
	}

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
		return false
	}
	if user == nil {
		fmt.Println(errorMsg("Invalid credentials."))
		return false
	}

	userID = user.ID
	fmt.Println(successMsg("Login successful!"))
	return true
}

func handleSpecialCommands(input string, successMsg func(a ...interface{}) string) bool {
	switch input {
	case "exit":
		fmt.Println(successMsg("Goodbye!"))
		os.Exit(0)
		return false // unreachable, but satisfies compiler
	case "logout":
		userID = uuid.Nil
		fmt.Println(successMsg("Logged out."))
		return true
	default:
		return false
	}
}

func handleCommand(args []string, scv *account.Service, errorMsg, successMsg func(a ...interface{}) string) {
	cmd := args[0]

	switch cmd {
	case "create":
		handleCreateAccount(scv, errorMsg, successMsg)
	case "deposit":
		handleDeposit(args, scv, errorMsg, successMsg)
	case "withdraw":
		handleWithdraw(args, scv, errorMsg, successMsg)
	case "balance":
		handleBalance(args, scv, errorMsg, successMsg)
	default:
		fmt.Println(errorMsg("Unknown command:"), cmd)
	}
}

func handleCreateAccount(scv *account.Service, errorMsg, successMsg func(a ...interface{}) string) {
	a, err := scv.CreateAccount(context.Background(), dto.AccountCreate{UserID: userID})
	if err != nil {
		fmt.Println(errorMsg("Error creating a:"), err)
		return
	}

	balance, err := scv.GetBalance(userID, a.ID)
	if err != nil {
		fmt.Println(errorMsg("Error fetching a balance:"), err)
		return
	}

	fmt.Println(successMsg(fmt.Sprintf("Account created: ID=%s, Balance=%.2f", a.ID, balance)))
}

func handleDeposit(args []string, scv *account.Service, errorMsg, successMsg func(a ...interface{}) string) {
	if len(args) < 3 {
		fmt.Println(errorMsg("Usage: deposit <account_id> <amount>"))
		return
	}

	accountID := args[1]
	amount, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		fmt.Println(errorMsg("Invalid amount:"), err)
		return
	}

	err = scv.Deposit(context.Background(), commands.DepositCommand{})
	if err != nil {
		fmt.Println(errorMsg("Error depositing:"), err)
		return
	}

	fmt.Println(successMsg(fmt.Sprintf("Deposited %.2f to account %s", amount, accountID)))
}

func handleWithdraw(args []string, scv *account.Service, errorMsg, successMsg func(a ...interface{}) string) {
	if len(args) < 3 {
		fmt.Println(errorMsg("Usage: withdraw <account_id> <amount>"))
		return
	}

	accountID := args[1]
	amount, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		fmt.Println(errorMsg("Invalid amount:"), err)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Bank Account Number (leave blank if not applicable): ")
	bankAccountNumber, _ := reader.ReadString('\n')
	bankAccountNumber = strings.TrimSpace(bankAccountNumber)

	fmt.Print("Routing Number (leave blank if not applicable): ")
	routingNumber, _ := reader.ReadString('\n')
	routingNumber = strings.TrimSpace(routingNumber)

	fmt.Print("External Wallet Address (leave blank if not applicable): ")
	externalWalletAddress, _ := reader.ReadString('\n')
	externalWalletAddress = strings.TrimSpace(externalWalletAddress)

	err = scv.Withdraw(context.Background(), commands.WithdrawCommand{
		UserID:    userID,
		AccountID: uuid.MustParse(accountID),
		Amount:    amount,
		Currency:  "USD",
		ExternalTarget: &commands.ExternalTarget{
			BankAccountNumber:     bankAccountNumber,
			RoutingNumber:         routingNumber,
			ExternalWalletAddress: externalWalletAddress,
		},
	})
	if err != nil {
		fmt.Println(errorMsg("Error withdrawing:"), err)
		return
	}

	fmt.Println(successMsg(fmt.Sprintf("Withdrew %.2f from account %s", amount, accountID)))
}

func handleBalance(args []string, scv *account.Service, errorMsg, successMsg func(a ...interface{}) string) {
	if len(args) < 2 {
		fmt.Println(errorMsg("Usage: balance <account_id>"))
		return
	}

	accountID := args[1]
	balance, err := scv.GetBalance(userID, uuid.MustParse(accountID))
	if err != nil {
		fmt.Println(errorMsg("Error fetching balance:"), err)
		return
	}

	fmt.Println(successMsg(fmt.Sprintf("Account %s balance: %.2f", accountID, balance)))
}
