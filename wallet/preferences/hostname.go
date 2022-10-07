package preferences

import "fmt"

// ConnectionApproval defines the type of log that is sent to the user.
type ConnectionApproval string

var (
	ApprovedOnlyThisTime ConnectionApproval = "APPROVED_ONLY_THIS_TIME"
	RejectedOnlyThisTime ConnectionApproval = "REJECTED_ONLY_THIS_TIME"
)

func ParseConnectionApproval(s string) (ConnectionApproval, error) {
	switch s {
	case "APPROVED_ONLY_THIS_TIME":
		return ApprovedOnlyThisTime, nil
	case "REJECTED_ONLY_THIS_TIME":
		return RejectedOnlyThisTime, nil
	default:
		return "", fmt.Errorf("the connection approval of type %q is not supported", s)
	}
}
