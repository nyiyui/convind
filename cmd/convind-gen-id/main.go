package main

import (
	"fmt"

	"inaba.kiyuri.ca/2025/convind/data"
)

func main() {
	id := data.GenerateRandomID()
	fmt.Println(id.String())
}
