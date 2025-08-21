package money

import (
	"fmt"
	"log"
)

// ExampleNew demonstrates how to create a new Money instance
func ExampleNew() {
	// Create money with USD
	usdMoney, err := New(100.50, Currency{Code: "USD", Decimals: 2})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("USD Money: %s\n", usdMoney.String())

	// Create money with EUR
	eurMoney, err := New(75.25, Currency{Code: "EUR", Decimals: 2})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("EUR Money: %s\n", eurMoney.String())

	// Create money with JPY (0 decimals)
	jpyMoney, err := New(1000, Currency{Code: "JPY", Decimals: 0})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("JPY Money: %s\n", jpyMoney.String())
	// Output:
	// USD Money: 100.50 USD
	// EUR Money: 75.25 EUR
	// JPY Money: 1000 JPY
}

// ExampleMoney_Add demonstrates adding money values
func ExampleMoney_Add() {
	// Create two USD amounts
	money1, _ := New(100.50, Currency{Code: "USD", Decimals: 2})
	money2, _ := New(25.75, Currency{Code: "USD", Decimals: 2})

	// Add them together
	result, err := money1.Add(money2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %s\n", result.String())
	// Output:
	// Result: 126.25 USD
}

// ExampleMoney_Subtract demonstrates subtracting money values
func ExampleMoney_Subtract() {
	// Create two USD amounts
	money1, _ := New(100.50, Currency{Code: "USD", Decimals: 2})
	money2, _ := New(25.75, Currency{Code: "USD", Decimals: 2})

	// Subtract them
	result, err := money1.Subtract(money2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %s\n", result.String())
	// Output:
	// Result: 74.75 USD
}

// ExampleMoney_Multiply demonstrates multiplying money by a factor
func ExampleMoney_Multiply() {
	// Create a USD amount
	money, _ := New(100.50, Currency{Code: "USD", Decimals: 2})

	// Multiply by 2.5
	result, err := money.Multiply(2.5)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %s\n", result.String())
	// Output:
	// Result: 251.25 USD
}

// ExampleMoney_Divide demonstrates dividing money by a factor
func ExampleMoney_Divide() {
	// Create a USD amount
	money, _ := New(100.50, Currency{Code: "USD", Decimals: 2})

	// Divide by 2
	result, err := money.Divide(2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %s\n", result.String())
	// Output:
	// Result: 50.25 USD
}

// ExampleMoney_Comparison demonstrates comparing money values
func ExampleMoney_comparison() {
	// Create two USD amounts
	money1, _ := New(100.50, Currency{Code: "USD"})
	money2, _ := New(75.25, Currency{Code: "USD"})

	// Compare them
	greater, _ := money1.GreaterThan(money2)
	equal := money1.Equals(money2)
	less, _ := money1.LessThan(money2)

	fmt.Printf("Greater than: %t\n", greater)
	fmt.Printf("Equal: %t\n", equal)
	fmt.Printf("Less than: %t\n", less)
	// Output:
	// Greater than: true
	// Equal: false
	// Less than: false
}

// ExampleMoney_IsPositive demonstrates checking if money is positive
func ExampleMoney_IsPositive() {
	// Create positive and negative amounts
	positive, _ := New(100.50, Currency{Code: "USD"})
	negative, _ := New(-25.75, Currency{Code: "USD"})
	zero, _ := New(0, Currency{Code: "USD"})

	fmt.Printf("Positive amount is positive: %t\n", positive.IsPositive())
	fmt.Printf("Negative amount is positive: %t\n", negative.IsPositive())
	fmt.Printf("Zero amount is positive: %t\n", zero.IsPositive())
	// Output:
	// Positive amount is positive: true
	// Negative amount is positive: false
	// Zero amount is positive: false
}

// ExampleMoney_IsZero demonstrates checking if money is zero
func ExampleMoney_IsZero() {
	// Create positive and zero amounts
	positive, _ := New(100.50, Currency{Code: "USD"})
	zero, _ := New(0, Currency{Code: "USD"})

	fmt.Printf("Positive amount is zero: %t\n", positive.IsZero())
	fmt.Printf("Zero amount is zero: %t\n", zero.IsZero())
	// Output:
	// Positive amount is zero: false
	// Zero amount is zero: true
}

// ExampleMoney_Currency demonstrates getting the currency
func ExampleMoney_Currency() {
	// Create money with different currencies
	usdMoney, _ := New(100.50, Currency{Code: "USD"})
	eurMoney, _ := New(75.25, Currency{Code: "EUR"})
	jpyMoney, _ := New(1000, Currency{Code: "JPY"})

	fmt.Printf("USD Money currency: %s\n", usdMoney.Currency())
	fmt.Printf("EUR Money currency: %s\n", eurMoney.Currency())
	fmt.Printf("JPY Money currency: %s\n", jpyMoney.Currency())
	// Output:
	// USD Money currency: USD
	// EUR Money currency: EUR
	// JPY Money currency: JPY
}

// ExampleMoney_Amount demonstrates getting the amount
func ExampleMoney_Amount() {
	// Create a USD amount
	usdMoney, _ := New(100.50, Currency{Code: "USD", Decimals: 2})
	fmt.Printf("USD amount: %d\n", usdMoney.Amount())

	// Create a JPY amount
	jpyMoney, _ := New(1000, Currency{Code: "JPY", Decimals: 0})
	fmt.Printf("JPY amount: %d\n", jpyMoney.Amount())
	// Output:
	// USD amount: 10050
	// JPY amount: 1000
}

// ExampleMoney_AmountFloat demonstrates getting the amount as float64
func ExampleMoney_AmountFloat() {
	// Create a USD amount
	usdMoney, _ := New(100.50, Currency{Code: "USD", Decimals: 2})
	fmt.Printf("USD amount float: %.2f\n", usdMoney.AmountFloat())

	// Create a JPY amount
	jpyMoney, _ := New(1000, Currency{Code: "JPY", Decimals: 0})
	fmt.Printf("JPY amount float: %.0f\n", jpyMoney.AmountFloat())
	// Output:
	// USD amount float: 100.50
	// JPY amount float: 1000
}

// ExampleNewFromSmallestUnit demonstrates creating money from smallest unit
func ExampleNewFromSmallestUnit() {
	// Create USD from cents (10050 cents = $100.50)
	usdMoney, err := NewFromSmallestUnit(10050, Currency{Code: "USD", Decimals: 2})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("USD from cents: %s\n", usdMoney.String())

	// Create JPY from yen (1000 yen = ¥1000)
	jpyMoney, err := NewFromSmallestUnit(1000, Currency{Code: "JPY", Decimals: 0})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("JPY from yen: %s\n", jpyMoney.String())
	// Output:
	// USD from cents: 100.50 USD
	// JPY from yen: 1000 JPY
}

// ExampleMoney_String demonstrates string representation
func ExampleMoney_String() {
	// Create different currency amounts
	usdMoney, _ := New(100.50, Currency{Code: "USD", Decimals: 2})
	eurMoney, _ := New(75.25, Currency{Code: "EUR", Decimals: 2})
	jpyMoney, _ := New(1000, Currency{Code: "JPY", Decimals: 0})

	// Print string representation
	fmt.Printf("USD: %s\n", usdMoney.String())
	fmt.Printf("EUR: %s\n", eurMoney.String())
	fmt.Printf("JPY: %s\n", jpyMoney.String())
	// Output:
	// USD: 100.50 USD
	// EUR: 75.25 EUR
	// JPY: 1000 JPY
}
