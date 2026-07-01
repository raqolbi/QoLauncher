package supervisor

import (
	"os"
	"os/signal"
	"syscall"
)

func registerSignals(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
}
