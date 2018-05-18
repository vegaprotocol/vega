package services

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

type OrderService interface {
	CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error)
}

type Order struct {
	Market    string
	Party     string
	Side      int32
	Price     uint64
	Size      uint64
	Remaining uint64
	Timestamp uint64
	Type      int
}

func NewOrder(
	market string,
	party string,
	side int32,
	price uint64,
	size uint64,
	remaining uint64,
	timestamp uint64,
	tradeType int,
) Order {
	return Order {
		market,
		party,
		side,
		price,
		size,
		remaining,
		timestamp,
		tradeType,
	}
}

func (o *Order) Json() ([]byte, error) {
	return json.Marshal(o)
}

func (o *Order) JsonWithEncoding() (string, error) {
	json, err := o.Json()
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(json)
	return encoded, err
}

type rpcOrderService struct {
}

func NewRpcOrderService() OrderService {
	return &rpcOrderService{}
}

func (p *rpcOrderService) CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error) {

	order := NewOrder(market, party, side, price, size, size, unixTimestamp(time.Now().UTC()), 0)
	payload, err := order.JsonWithEncoding()
	if err != nil {
		return false, err
	}

	var reqUrl = "http://localhost:46657/broadcast_tx_commit?tx=%22" + newGuid() + "|" + payload + "%22"
	resp, err := http.Get(reqUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// For debugging only
	// body, err := ioutil.ReadAll(resp.Body)
	//if err == nil {
	//	fmt.Println(string(body))
	//}

	return true, err
}