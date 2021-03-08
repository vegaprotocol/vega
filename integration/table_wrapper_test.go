package core_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog/gherkin"
)

type TableWrapper gherkin.DataTable

func (t TableWrapper) Parse() []RowWrapper {
	dt := gherkin.DataTable(t)
	out := make([]RowWrapper, 0, len(dt.Rows)-1)

	for _, row := range dt.Rows[1:] {
		wrapper := RowWrapper{values: map[string]string{}}
		for i := range row.Cells {
			wrapper.values[dt.Rows[0].Cells[i].Value] = row.Cells[i].Value
		}
		out = append(out, wrapper)
	}

	return out
}

type RowWrapper struct {
	values map[string]string
}

func (r RowWrapper) Str(name string) string {
	return r.values[name]
}

func (r RowWrapper) StrSlice(name, sep string) []string {
	return strings.Split(r.values[name], sep)
}

func (r RowWrapper) U64(name string) (uint64, error) {
	rawValue := r.values[name]
	return strconv.ParseUint(rawValue, 10, 0)
}

func (r RowWrapper) U64Slice(name, sep string) ([]uint64, error) {
	rawValue := r.values[name]
	if len(rawValue) == 0 {
		return []uint64{}, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]uint64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseUint(rawValues[i], 10, 0)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) I64(name string) (int64, error) {
	rawValue := r.values[name]
	return strconv.ParseInt(rawValue, 10, 0)
}

func (r RowWrapper) I64Slice(name, sep string) ([]int64, error) {
	rawValue := r.values[name]
	if len(rawValue) == 0 {
		return []int64{}, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]int64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseInt(rawValues[i], 10, 0)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) F64(name string) (float64, error) {
	rawValue := r.values[name]
	return strconv.ParseFloat(rawValue, 10)
}

func (r RowWrapper) F64Slice(name, sep string) ([]float64, error) {
	rawValue := r.values[name]
	if len(rawValue) == 0 {
		return nil, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]float64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseFloat(rawValues[i], 10)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) Bool(name string) (bool, error) {
	rawValue := r.values[name]
	if rawValue == "true" {
		return true, nil
	} else if rawValue == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value: %v", name)
}
