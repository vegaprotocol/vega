package datastore

// Trade
type Trade struct {
	ID        string
	Market    string
	Price     uint64
	Size      uint64
	Buyer     string
	Seller    string
	Side      int32
	Timestamp uint64

	OrderID   string
	//Order     *Order
}

type Order struct {
	ID        string
	Market    string
	Party     uint64
	Side      int32
	Price     uint64
	Size      uint64
	Type      int32
	Timestamp uint64
}

type TradeStore interface {
	// Get retrieves a trade for a given id.
	Get(id string) (*Trade, error)
	// FindByOrderId retrieves all trades for a given order id.
	FindByOrderId(orderId string) ([]*Trade, error)
	// Put stores a trade.
	Put(r *Trade) error
	// Removes a trade from the store.
	Delete(r *Trade) error
}

type OrderStore interface {
	// Get retrieves an order for a given id.
	Get(id string) (*Order, error)
	// FindByParty retrieves all order for a given party name.
	//FindByParty(party string) ([]*Order, error)
	// Put stores a trade.
	Put(r *Order) error
	// Removes a trade from the store.
	Delete(r *Order) error
}


// We could have one large store interface
//type Store interface {
//	// Get retrieves an order for a given id.
//	GetTrade(id string) (*Order, error)
//	// FindByParty retrieves all order for a given party name.
//	//FindByParty(party string) ([]*Order, error)
//	// Put stores a trade.
//	PutTrade(r *Order) error
//	// Removes a trade from the store.
//	DeleteTrade(r *Order) error
//	// FindByOrderId retrieves all trades for a given order id.
//	FindTradesByOrderId(orderId string) ([]*Trade, error)
//	// Get retrieves an order for a given id.
//	GetOrder(id string) (*Order, error)
//	// FindByParty retrieves all order for a given party name.
//	//FindByParty(party string) ([]*Order, error)
//	// Put stores a trade.
//	PutOrder(r *Order) error
//	// Removes a trade from the store.
//	DeleteOrder(r *Order) error
//}


