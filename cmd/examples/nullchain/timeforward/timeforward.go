package timefoward

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	config "code.vegaprotocol.io/vega/cmd/examples/nullchain/config"
)

func move(raw string) error {
	values := map[string]string{"forward": raw}

	jsonValue, _ := json.Marshal(values)

	r, err := http.Post(config.TimeforwardAddress, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	if r.StatusCode == http.StatusOK {
		return nil
	}

	data, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(data))
	return err
}

func MoveByDuration(d time.Duration) error {
	return move(d.String())
}

func MoveToDate(t time.Time) error {
	return move(t.Format(time.RFC3339))
}
