package print

import (
	"encoding/json"
	"fmt"
)

func PrintPretty(v interface{}) {

	b, err := json.MarshalIndent(v, "", "   ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))
}
