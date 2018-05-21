package orders

import (
	"net/http"
	"time"
)

type OrderService interface {
	CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error)
}

type rpcOrderService struct {
}

func NewRpcOrderService() OrderService {
	return &rpcOrderService{}
}

func (p *rpcOrderService) CreateOrder(market string, party string, side int32, price uint64, size uint64) (success bool, err error) {

	// todo bind json / Gin
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