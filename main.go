package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/kardianos/service"
)

// logger is a variable of type service.Logger.
// It is used to log messages and errors within the program.
// The logger is initialized and assigned a value when starting the program.
// It is used to record various events such as service installation, service startup, and program execution.
// The logger provides methods like Info, Error, and Errorf to log different types of messages and errors.
var logger service.Logger

// program represents a program structure.
//
//	Define Start and Stop methods.
type program struct {
	exit chan struct{}
	s    service.Service
}

// Start starts the service and performs necessary setup tasks.
//
// If running in interactive mode, it checks the status of the service and performs the following actions:
//   - If the service is not installed, it installs the service.
//   - If the service is installed but not started, it starts the service.
//   - If the service is already running, it simply displays a message indicating that the service is running.
//   - If the service has any other status, it logs an error and exits the program.
//
// If running under a service manager, it logs a message indicating that it is running under the service manager.
//
// It creates an exit channel and starts a goroutine to run the actual work asynchronously.
//
// This method does not block and returns nil if the service starts successfully.
// Otherwise, it returns an error.
func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
		for {
			switch status, err := s.Status(); status {
			case 0:
				if errors.Is(err, service.ErrNotInstalled) {
					err = s.Install()
					if err != nil {
						return err
					}
				} else {
					return err
				}
				logger.Info("Service installed")
				fallthrough
			case 2:
				err = s.Start()
				if err != nil {
					return err
				}
				logger.Info("Service started")
				fallthrough
			case 1:
				logger.Info("Service is running")
			default:
				logger.Errorf("Unexpected status from service: %v", status)
				os.Exit(1)
			}
			os.Exit(0)
		}
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

// run executes the main logic of the program, continuously logging a message every 2 seconds while running.
// It retrieves the platform the program is running on and logs an informational message.
// A ticker is created to fire a signal every 2 seconds.
// In a loop, the function waits for either a tick from the ticker or a signal on the p.exit channel.
// When a tick is received, it logs an informational message indicating that it is still running at the respective time.
// When a signal is received on the p.exit channel, the function returns nil to indicate successful execution.
// This function is executed in a separate goroutine, allowing the program to continue running concurrently without blocking.
// It does not return any errors.
func (p *program) run() error {
	logger.Infof("I'm running %v.", service.Platform())
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case tm := <-ticker.C:
			logger.Infof("Still running at %v...", tm)
		case <-p.exit:
			return nil
		}
	}
}

// Stop stops the program execution.
// Any work in Stop should be quick, usually a few seconds at most.
func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logger.Info("I'm Stopping!")
	close(p.exit)
	return nil
}

// main is the entry point of the program.
//
// It parses the command line flags and initializes the service configuration.
// It creates a program instance and registers it as a service.
// It sets up the logger and starts listening to the errors channel.
// If the "-service" flag is provided, it calls the service control function
// with the provided action and exits the program.
// Otherwise, it starts the service and logs any errors that occur.
// The program exits when the service is stopped or an error is encountered.
func main() {
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	options := make(service.KeyValue)
	options["Restart"] = "always"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	options["OnFailure"] = "restart"
	svcConfig := &service.Config{
		Name:        "GoServiceExampleLogging",
		DisplayName: "Go Service Example for Logging",
		Description: "This is an example Go service that outputs log messages.",
		Option:      options,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	prg.s = s
	if err != nil {
		log.Fatal(err)
	}
	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		return
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
