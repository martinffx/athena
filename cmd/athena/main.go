package main

import (
	"log"

	"athena/internal"
)

func main() {
	if err := internal.Execute(); err != nil {
		log.Fatal(err)
	}
}
