package api

import (
	"time"
	"github.com/satori/go.uuid"
	"encoding/base64"
	"encoding/json"
)

func unixTimestamp(datetime time.Time) uint64 {
	return uint64(datetime.UnixNano() / (int64(time.Millisecond)/int64(time.Nanosecond)))
}

func newGuid() string {
	return uuid.NewV4().String()
}

func jsonWithEncoding(o interface{}) (string, error) {
	json, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(json)
	return encoded, err
}
