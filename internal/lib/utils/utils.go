package utils

import (
	"encoding/json"
	"fmt"
)

// To take a quick look at the contents of a slice, struct, or map
func PrintJson(v interface{}) {
	json, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
		return
	}
	fmt.Println("JSON:", string(json))
}
