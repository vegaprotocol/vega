package execution

// GetPeggedOrderCount returns the number of pegged orders in the market
func (m *Market) GetPeggedOrderCount() int {
	return len(m.peggedOrders)
}

// GetParkedOrderCount returns hte number of parked orders in the market
func (m *Market) GetParkedOrderCount() int {
	return len(m.parkedOrders)
}

// GetPeggedExpiryOrderCount returns the number of pegged order that can expire
func (m *Market) GetPeggedExpiryOrderCount() int {
	return m.expiringPeggedOrders.GetExpiryingOrderCount()
}
