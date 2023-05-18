package main

import (
	"fmt"
	"os"
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

func (pp *PersistenceProfiler) QueryUnseenMessages() ([]string, error) {
	logger := PerfLogger{
		csvFile: pp.csvFile,
		apiName: "QueryUnseenMessages",
		start:   time.Now(),
	}
	defer logger.Complete()
	return pp.p.QueryUnseenMessages()
}

func (pp *PersistenceProfiler) InsertUnseenMessage() error {
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
		fmt.Println("could not write to csv", err)
	}
}
