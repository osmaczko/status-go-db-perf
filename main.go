package main

import (
	"flag"
	"log"
)

func main() {
	dbPath := flag.String("dbPath", "", "Path to the database")
	key := flag.String("key", "", `"0x" + lowercase(keccak256(clearPassword))`)
	flag.Parse()

	persistence, err := NewPersistence(*dbPath, *key)
	if err != nil {
		log.Println(err)
		return
	}
	defer persistence.Close()

	ids, err := persistence.QueryUnseenMessages()
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(ids)
}
