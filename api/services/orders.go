package services

import (
"encoding/json"
"encoding/base64"
"net/http"
"time"
"io/ioutil"
"fmt"
)

type OrderService interface {
	CreateOrder(market string, party string, side int32, price uint64, size uint64) string
}


type Order struct {
	Market string
	Party string
	Side int32
	Price uint64
	Size uint64
	Remaining uint64
	Type int
	Timestamps string
}

func NewOrder(
	market		string,
	party		string,
	side		int32,
	price		uint64,
	size		uint64,
	remaining	uint64,
	tradeType	int,
	timestamps	string,
) Order {
	return Order{}
}


type RpcOrderService struct {
}

func NewRpcOrderService() RpcOrderService {
	return RpcOrderService{}
}

func (p *RpcOrderService) CreateOrder(market string, party string, side int32, price uint64, size uint64) string {

	tradeOrder := NewOrder(market, party, side, price, size, size,0, time.Now().String())
	json, _ := json.Marshal(tradeOrder)
	payload := base64.StdEncoding.EncodeToString(json)

	resp, err := http.NewRequest(http.MethodGet, "http://localhost:46657/broadcast_tx_commit?tx=" + payload, nil)

	if err != nil {
		return "RPC::CREATE::ORDER::FAILURE"
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	fmt.Println(body)

	return "RPC::CREATE::ORDER::SUCCESS"
}



/*

export function placeOrder (order: CurrentOrder, market: string) {
  const mutation = {
    Market: market,
    Party: guid,
    Side: order.side === 'BUY' ? 0 : 1,
    Price: order.price,
    Size: order.size,
    Remaining: order.size,
    Type: 0,
    Timestamps: Date.now()
  }

  const encodedMutation = btoa(JSON.stringify(mutation))
  // @ts-ignore
  let host = window.host
  let baseUrl = globalStore.getState().currentServer
  let req = `http://${baseUrl}:46657/broadcast_tx_commit?tx="${Math.random()}|${encodedMutation}"`

  fetch(req)
    .then(() => {
      globalStore.dispatch(actions.completeCurrentOrder())
    })
    .catch(console.error)
}

 */