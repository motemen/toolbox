package main

import (
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
)

func main() {
	m, err := mail.ReadMessage(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	if len(m.Header["Message-Id"]) == 0 {
		os.Exit(2)
	}

	for _, mid := range m.Header["Message-Id"] {
		mid = strings.TrimSpace(mid)
		mid = strings.TrimLeft(mid, "<")
		mid = strings.TrimRight(mid, ">")
		fmt.Println(mid)
	}
}
