package steps

import (
	"fmt"
	"time"
)

func MarketsStartOnAndExpireOn(startDate, expiryDate string) (string, string, error) {
	_, err := time.Parse("2006-01-02T15:04:05Z", startDate)
	if err != nil {
		return startDate, expiryDate, fmt.Errorf("invalid start date %v", err)
	}
	_, err = time.Parse("2006-01-02T15:04:05Z", expiryDate)
	if err != nil {
		return startDate, expiryDate, fmt.Errorf("invalid expiry date %v", err)
	}
	return startDate, expiryDate, nil
}
