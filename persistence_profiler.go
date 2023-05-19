package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

type PersistenceProfiler struct {
	p           *Persistence
	csvFile     *os.File
	csvFileLock sync.Mutex
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
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				time.Sleep(time.Duration(200+rand.Intn(500)) * time.Millisecond)
				defer wg.Done()
				if _, err := pp.queryUnseenMessages(); err != nil {
					errCh <- fmt.Errorf("queryUnseenMessages failed: %v", err)
				}
			}()
		}

		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)
				defer wg.Done()
				if err := pp.insertUnseenMessage(); err != nil {
					errCh <- fmt.Errorf("insertUnseenMessage failed: %v", err)
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
		csvFile:     pp.csvFile,
		csvFileLock: &pp.csvFileLock,
		apiName:     "QueryUnseenMessages",
		start:       time.Now(),
	}
	defer logger.Complete()
	return pp.p.QueryUnseenMessages()
}

func (pp *PersistenceProfiler) insertUnseenMessage() error {
	logger := PerfLogger{
		csvFile:     pp.csvFile,
		csvFileLock: &pp.csvFileLock,
		apiName:     "InsertUnseenMessage",
		start:       time.Now(),
	}
	defer logger.Complete()
	return pp.p.InsertUnseenMessage()
}

type PerfLogger struct {
	csvFile     *os.File
	csvFileLock *sync.Mutex
	apiName     string
	start       time.Time
}

func (pf *PerfLogger) Complete() {
	duration := time.Since(pf.start)

	line := fmt.Sprintf("%sµ%d\n", pf.apiName, duration.Nanoseconds())
	pf.csvFileLock.Lock()
	defer pf.csvFileLock.Unlock()
	if _, err := pf.csvFile.Write([]byte(line)); err != nil {
		log.Println("could not write to csv", err)
	}
}
