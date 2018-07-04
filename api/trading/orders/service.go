package orders

import (
	"net/http"
	"time"
	"vega/api/trading/orders/models"
)

type OrderService interface {
	CreateOrder(order models.Order) (success bool, err error)
}

type rpcOrderService struct {
}

func NewRpcOrderService() OrderService {
	return &rpcOrderService{}
}

func (p *rpcOrderService) CreateOrder(order models.Order) (success bool, err error) {
	
	// todo additional validation?
	utcNow := time.Now().UTC()
	order.Timestamp = unixTimestamp(utcNow)
	order.Remaining = order.Size

	payload, err := order.JsonWithEncoding()
	if err != nil {
		return false, err
	}

	reqUrl := "http://localhost:46657/broadcast_tx_async?tx=%22" + newGuid() + "|" + payload + "%22"
	client := &http.Client{Timeout:5 * time.Second}
	resp, err := client.Get(reqUrl)
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