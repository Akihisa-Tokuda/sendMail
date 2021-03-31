package main

import (
	"fmt"
	"log"
	"os"
	"sendMail/sender"

	"github.com/kardianos/service"
)

var logger service.Logger

type exarvice struct {
	exit chan struct{}
}

func (e *exarvice) run() error {

	logger.Info("Exarvice Start !!!")

	sender.StartObserve()
	return nil
}

func (e *exarvice) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	e.exit = make(chan struct{})

	go e.run()
	return nil
}

func (e *exarvice) Stop(s service.Service) error {
	close(e.exit)
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "CreatedFileSenderViaMail",
		DisplayName: "Created File Sender via Email",
		Description: "指定フォルダにファイルが作成されたらメール送信するサービス",
	}

	// Create Exarvice service
	program := &exarvice{}
	s, err := service.New(program, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Setup the logger
	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal()
	}

	if len(os.Args) > 1 {

		err = service.Control(s, os.Args[1])
		if err != nil {
			fmt.Printf("Failed (%s) : %s\n", os.Args[1], err)
			return
		}
		fmt.Printf("Succeeded (%s)\n", os.Args[1])
		return
	}

	// run in terminal
	s.Run()
}
