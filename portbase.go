package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Safing/portbase/info"
	"github.com/Safing/portbase/log"
	"github.com/Safing/portbase/modules"
	// include packages here
)

func main() {

	// Set Info
	info.Set("Portbase", "0.0.1")

	// Start
	err := modules.Start()
	if err != nil {
		if err == modules.ErrCleanExit {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Shutdown
	// catch interrupt for clean shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(
		signalCh,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	select {
	case <-signalCh:
		fmt.Println(" <INTERRUPT>")
		log.Warning("main: program was interrupted, shutting down.")
		modules.Shutdown()
	case <-modules.ShuttingDown():
	}

}
