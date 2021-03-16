package steps

import (
	"fmt"
)

func MarketsStartOnAndExpireOn(startDate, expiryDate string) (string, string, error) {
	_, err := Time(startDate)
	if err != nil {
		return startDate, expiryDate, fmt.Errorf("invalid start date %v", err)
	}
	_, err = Time(expiryDate)
	if err != nil {
		return startDate, expiryDate, fmt.Errorf("invalid expiry date %v", err)
	}
	return startDate, expiryDate, nil
}
