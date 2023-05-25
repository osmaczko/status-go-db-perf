package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
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

func profile(dbPath string, key string, maxOpenConns int, maxIdleConns int) (time.Duration, error) {
	tempDB, err := databaseTemp(dbPath)
	if err != nil {
		return 0, err
	}
	defer os.Remove(tempDB.Name())
	log.Println("Temporary database created: ", tempDB.Name())

	persistence, err := NewPersistence(tempDB.Name(), key, maxOpenConns, maxIdleConns)
	if err != nil {
		return 0, err
	}
	defer persistence.Cleanup()

	profiler, err := NewPersistenceProfiler(persistence)
	if err != nil {
		return 0, err
	}
	defer profiler.Cleanup()

	start := time.Now()
	if err := profiler.Perform(); err != nil {
		return 0, err
	}

	return time.Since(start), nil
}

func main() {
	dbPath := flag.String("dbPath", "", "Path to the database")
	key := flag.String("key", "", `"0x" + lowercase(keccak256(clearPassword))`)
	flag.Parse()

	csvFile, err := os.Create("output/perf-compare-connections-config" + fmt.Sprint(time.Now().Unix()) + ".csv")
	if err != nil {
		log.Println(err)
		return
	}
	defer csvFile.Close()

	for i := 1; i <= 20; i = i + 4 {
		for j := 1; j <= i; j = j + 4 {
			result, err := profile(*dbPath, *key, i, j)
			if err != nil {
				log.Println(err)
				return
			}
			line := fmt.Sprintf("%d %d %d\n", i, j, result.Milliseconds())
			if _, err := csvFile.Write([]byte(line)); err != nil {
				log.Println("could not write to csv", err)
			}
			log.Print(line)
		}
	}
}
