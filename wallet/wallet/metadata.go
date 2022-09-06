package wallet

const KeyNameMeta = "name"

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func GetKeyName(meta []Metadata) string {
	for _, m := range meta {
		if m.Key == KeyNameMeta {
			return m.Value
		}
	}

	return "<No name>"
}
