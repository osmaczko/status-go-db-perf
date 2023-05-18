package main

import (
	"flag"
	"io"
	"log"
	"os"
)

func databaseTemp(path string) (*os.File, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	tempDB, err := os.CreateTemp("", "status-go-db-perf-temp")
	if err != nil {
		return nil, err
	}

	sourceDB, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(tempDB, sourceDB); err != nil {
		return nil, err
	}

	sourceDB.Close()
	tempDB.Close()

	return tempDB, nil
}

func main() {
	dbPath := flag.String("dbPath", "", "Path to the database")
	key := flag.String("key", "", `"0x" + lowercase(keccak256(clearPassword))`)
	flag.Parse()

	tempDB, err := databaseTemp(*dbPath)
	if err != nil {
		log.Println(err)
		return
	}
	defer os.Remove(tempDB.Name())

	log.Println("Temporary database created: ", tempDB.Name())

	persistence, err := NewPersistence(tempDB.Name(), *key)
	if err != nil {
		log.Println(err)
		return
	}
	defer persistence.Cleanup()

	profiler, err := NewPersistenceProfiler(persistence)
	if err != nil {
		log.Println(err)
		return
	}
	defer profiler.Cleanup()

	if err := profiler.Perform(); err != nil {
		log.Println(err)
		return
	}
}
