package matching

import (
"fmt"
"math/rand"
"time"
)

type Trades []Trade

type Trade struct {
	price int
	timestamp int
	volume int
}

type Kandles []Kandle

type Kandle struct {
	max int
	min int
	volume int
}

func main() {
	trades := generateRandomTrades(20)
	fmt.Println("trades: ", trades)
	fmt.Println("kandles: ", trades.aggregate(3, 2))
}

func generateRandomTrades(n int) (trades Trades) {
	var timestamp int
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < n; i++ {
		if i!= 0 && i%3 == 0 {
			timestamp++
		}
		price := rand.Intn(40) + 80
		size := rand.Intn(400) + 800
		trades = append(trades, Trade{price, timestamp, size})
	}
	return trades
}

func (t Trades) aggregate(since, interval int) (kandles Kandles) {
	var currentTimestamp int
	var intervalProgression int

	var kandle Kandle

	for idx, trade := range t {
		if trade.timestamp < since {
			continue
		}
		fmt.Println("trade: ", trade)

		if currentTimestamp != trade.timestamp {
			fmt.Println("currentTimestamp ", currentTimestamp)
			currentTimestamp = trade.timestamp
			if kandle.volume != 0 {
				intervalProgression++
			}
		}

		if intervalProgression < interval {
			kandle.volume += trade.volume
			if kandle.max < trade.price {
				kandle.max = trade.price
			}
			if kandle.min > trade.price || kandle.min == 0 {
				kandle.min = trade.price
			}
			fmt.Println("updating kandle")
		}

		if idx == len(t)-1 {
			fmt.Println("end of trades")
			kandles = append(kandles, kandle)
			intervalProgression = 0
			fmt.Println("appending kandle")
			fmt.Println()
			kandle.volume = 0
			break
		}

		fmt.Printf("progression: %d/%d\n", intervalProgression, interval)
		fmt.Printf("current T: %d, next T %d \n", currentTimestamp, t[idx+1].timestamp)
		if intervalProgression+1 == interval  && t[idx+1].timestamp != currentTimestamp {
			kandles = append(kandles, kandle)
			intervalProgression = 0
			fmt.Println("appending kandle")
			fmt.Println()
			kandle.volume = 0
		}


	}
	return kandles
}