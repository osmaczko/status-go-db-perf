package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type PersistenceProfiler struct {
	p       *Persistence
	csvFile *os.File
}

func NewPersistenceProfiler(p *Persistence) (*PersistenceProfiler, error) {
	csvFile, err := os.Create("perf-" + fmt.Sprint(time.Now().Unix()) + ".csv")
	if err != nil {
		return nil, err
	}

	return &PersistenceProfiler{
		p:       p,
		csvFile: csvFile,
	}, nil
}

func (pp *PersistenceProfiler) Cleanup() {
	pp.csvFile.Close()
}

func (pp *PersistenceProfiler) Perform() error {
	// log exclusive query
	if _, err := pp.queryUnseenMessages(); err != nil {
		return err
	}

	// log exclusive insert
	if err := pp.insertUnseenMessage(); err != nil {
		return err
	}

	// log concurrent reading and writing
	wg := sync.WaitGroup{}
	errCh := make(chan error)
	go func() {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := pp.queryUnseenMessages(); err != nil {
					errCh <- err
				}
			}()
		}

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := pp.insertUnseenMessage(); err != nil {
					errCh <- err
				}
			}()
		}

		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			return err // return on first error
		}
	}

	return nil
}

func (pp *PersistenceProfiler) queryUnseenMessages() ([]string, error) {
	logger := PerfLogger{
		csvFile: pp.csvFile,
		apiName: "QueryUnseenMessages",
		start:   time.Now(),
	}
	defer logger.Complete()
	return pp.p.QueryUnseenMessages()
}

func (pp *PersistenceProfiler) insertUnseenMessage() error {
	logger := PerfLogger{
		csvFile: pp.csvFile,
		apiName: "InsertUnseenMessage",
		start:   time.Now(),
	}
	defer logger.Complete()
	return pp.p.InsertUnseenMessage()
}

type PerfLogger struct {
	csvFile *os.File
	apiName string
	start   time.Time
}

func (pf *PerfLogger) Complete() {
	duration := time.Since(pf.start)

	line := fmt.Sprintf("%sµ%d\n", pf.apiName, duration.Nanoseconds())
	if _, err := pf.csvFile.Write([]byte(line)); err != nil {
		log.Println("could not write to csv", err)
	}
}
