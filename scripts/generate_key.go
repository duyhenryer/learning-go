package main

import (
	"fmt"
	"github.com/duyhenryer/go-rest-api/pkg/auth"
)

func main() {
	fmt.Println(auth.GenerateRandomKey())
}
