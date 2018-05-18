package services

import (
"encoding/json"
"encoding/base64"
"net/http"
"time"
"io/ioutil"
"fmt"
"github.com/satori/go.uuid"
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
	market    string,
	party     string,
	side      int32,
	price     uint64,
	size      uint64,
	remaining uint64,
	timestamp uint64,
	tradeType int,
) Order {
	return Order{}
}

type RpcOrderService struct {
}

func NewRpcOrderService() RpcOrderService {
	return RpcOrderService{}
}

func (p *RpcOrderService) CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error) {

	tradeOrder := NewOrder(market, party, side, price, size, size, unixTimestamp(time.Now().UTC()), 0)
	
	json, err := json.Marshal(tradeOrder)
	if err != nil {
		return false, err
	}

	payload := base64.StdEncoding.EncodeToString(json)
	seed := uuid.NewV4().String()
	resp, err := http.NewRequest(http.MethodGet, "http://localhost:46657/broadcast_tx_commit?tx=" + seed + "|" + payload, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	fmt.Println(body)

	return true, nil
}
