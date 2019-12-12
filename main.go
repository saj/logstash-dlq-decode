package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/saj/logstash-dlq-decode/internal/dlq"
)

func init() {
	log.SetFlags(0)
}

func main() {
	var (
		o   = bufio.NewWriter(os.Stdout)
		enc = json.NewEncoder(o)
		dec = dlq.NewDecoder(os.Stdin)
		n   = 1
	)
	for dec.Scan() {
		e, err := dec.Event()
		if err != nil {
			log.Fatalf("record %d: %v", n, err)
		}
		if err := enc.Encode(e); err != nil {
			log.Fatal(err)
		}
		n++
	}
	if err := dec.Err(); err != nil {
		log.Fatalf("record %d: %v", n, err)
	}
	if err := o.Flush(); err != nil {
		log.Fatal(err)
	}
}
