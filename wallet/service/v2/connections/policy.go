package connections

import "time"

type connectionPolicy interface {
	HasConnectionExpired(now time.Time) bool
	UpdateActivityDate(time.Time)
	IsLongLivingConnection() bool
	SetAsForcefullyClose()
	IsClosed() bool
}

// sessionPolicy is the policy to apply between a third-party application and a wallet.
// This type of policy is used for connections that only last as long as the service
// is live or until they are explicitly stopped.
// In short, a session connection doesn't survive the reboot of the service.
type sessionPolicy struct {
	expiryDate time.Time
	closed     bool
}

func (p *sessionPolicy) UpdateActivityDate(t time.Time) {
	p.expiryDate = t.Add(1 * time.Hour)
}

func (p *sessionPolicy) HasConnectionExpired(t time.Time) bool {
	return p.expiryDate.Before(t)
}

func (p *sessionPolicy) IsLongLivingConnection() bool {
	return false
}

func (p *sessionPolicy) SetAsForcefullyClose() {
	p.closed = true
}

func (p *sessionPolicy) IsClosed() bool {
	return p.closed
}

type longLivingConnectionPolicy struct {
	// expirationDate is an optional expiry date for this connection.
	expirationDate *time.Time
	closed         bool
}

func (p *longLivingConnectionPolicy) UpdateActivityDate(_ time.Time) {}

func (p *longLivingConnectionPolicy) HasConnectionExpired(now time.Time) bool {
	return p.expirationDate != nil && !p.expirationDate.After(now)
}

func (p *longLivingConnectionPolicy) IsLongLivingConnection() bool {
	return true
}

func (p *longLivingConnectionPolicy) SetAsForcefullyClose() {
	p.closed = true
}

func (p *longLivingConnectionPolicy) IsClosed() bool {
	return p.closed
}
