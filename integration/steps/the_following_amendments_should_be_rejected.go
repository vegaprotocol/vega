package steps

import (
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/helpers"
)

func TheFollowingAmendmentsShouldBeRejected(
	errHandler *helpers.ErrorHandler,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		party := row.MustStr("trader")
		reference := row.MustStr("reference")
		errMessage := row.MustStr("error")

		select {
		case <-time.After(2 * time.Second):
			return fmt.Errorf("couldn't find any rejected order amendment")
		case e := <-errHandler.ErrCh():
			switch et := e.(type) {
			case OrderAmendmentError:
				err := asOrderAmendmentError(e)
				if !isExpectedRejectedOrderAmendment(err, party, reference, errMessage) {
					return errUnexpectedRejectedOrderAmendment(err, party, reference, errMessage)
				}
				continue
			default:
				return fmt.Errorf("expecting error of type OrderAmendmentError but got %v", et)
			}
		}
	}
	return nil
}

func asOrderAmendmentError(e error) *OrderAmendmentError {
	err := &OrderAmendmentError{}
	errors.As(e, err)
	return err
}

func isExpectedRejectedOrderAmendment(err *OrderAmendmentError, party string, reference string, errMessage string) bool {
	return err.Err.Error() == errMessage &&
		err.OrderReference == reference
}

func errUnexpectedRejectedOrderAmendment(err *OrderAmendmentError, party string, reference string, errMessage string) error {
	return formatDiff(
		fmt.Sprintf("rejected amendment does not match for party \"%s\"", party),
		map[string]string{
			"order": reference,
			"error": errMessage,
		},
		map[string]string{
			"order": err.OrderReference,
			"error": err.Err.Error(),
		},
	)
}
