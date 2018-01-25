package main

import (
	"mbook"
)

func main() {
	book := mbook.NewBook("testMarket")
	book.AddOrder(mbook.Buy, 50, 105, "A")
	book.AddOrder(mbook.Buy, 125, 102, "B")
	book.AddOrder(mbook.Sell, 700, 100, "C")
}
