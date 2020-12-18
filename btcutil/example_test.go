package btcutil_test

import (
	"fmt"
	"math"

	"github.com/pkt-cash/pktd/btcutil"
)

func ExampleAmount() {
	a := btcutil.Amount(0)
	fmt.Println("Zero Satoshi:", a)

	a = btcutil.Amount(1e8)
	fmt.Println("100,000,000 Satoshis:", a)

	a = btcutil.Amount(1e5)
	fmt.Println("100,000 Satoshis:", a)
	// Output:
	// Zero Satoshi: 0 BTC
	// 100,000,000 Satoshis: 1 BTC
	// 100,000 Satoshis: 0.001 BTC
}

func ExampleNewAmount() {
	amountOne, err := btcutil.NewAmount(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountOne) // Output 1

	amountFraction, err := btcutil.NewAmount(0.01234567)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountFraction) // Output 2

	amountZero, err := btcutil.NewAmount(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountZero) // Output 3

	amountNaN, err := btcutil.NewAmount(math.NaN())
	if err != nil {
		fmt.Println(err.Message())
		return
	}
	fmt.Println(amountNaN) // Output 4

	// Output: 1 BTC
	// 0.01234567 BTC
	// 0 BTC
	// invalid bitcoin amount
}

func formatOrDie(a btcutil.Amount, to string) string {
	x, e := a.Format(to)
	if e != nil {
		panic(fmt.Sprintf("formatOrDie(%v) %v", to, e))
	}
	return x
}

func ExampleAmount_unitConversions() {
	amount := btcutil.Amount(44433322211100)

	fmt.Println("Satoshi to kBTC:", formatOrDie(amount, "kBTC"))
	fmt.Println("Satoshi to BTC:", amount)
	fmt.Println("Satoshi to MilliBTC:", formatOrDie(amount, "mBTC"))
	fmt.Println("Satoshi to MicroBTC:", formatOrDie(amount, "uBTC"))
	fmt.Println("Satoshi to Satoshi:", formatOrDie(amount, "Satoshi"))

	// Output:
	// Satoshi to kBTC: 444.333222111 kBTC
	// Satoshi to BTC: 444333.222111 BTC
	// Satoshi to MilliBTC: 444333222.111 mBTC
	// Satoshi to MicroBTC: 444333222111 μBTC
	// Satoshi to Satoshi: 44433322211100 Satoshi
}

func ExampleAmount_unitConversions1() {
	amount := btcutil.Amount(44433322211133)

	fmt.Println("Satoshi to kBTC:", formatOrDie(amount, "kBTC"))
	fmt.Println("Satoshi to BTC:", amount)
	fmt.Println("Satoshi to MilliBTC:", formatOrDie(amount, "mBTC"))
	fmt.Println("Satoshi to MicroBTC:", formatOrDie(amount, "uBTC"))
	fmt.Println("Satoshi to Satoshi:", formatOrDie(amount, "Satoshi"))

	// Output:
	// Satoshi to kBTC: 444.33322211133 kBTC
	// Satoshi to BTC: 444333.22211133 BTC
	// Satoshi to MilliBTC: 444333222.11133 mBTC
	// Satoshi to MicroBTC: 444333222111.33 μBTC
	// Satoshi to Satoshi: 44433322211133 Satoshi
}
