package fmt

import "fmt"

func PrettyPrint(data map[string]string) {
	for k, v := range data {
		fmt.Printf("%s:\n%s\n", k, v)
	}
}
