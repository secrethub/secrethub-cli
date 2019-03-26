// progress provides a printer that writes dots at a configured interval.
package progress

import (
	"fmt"
	"io"
	"time"
)

// Printer outputs a dot (.) every configured interval until Stop is called.
type Printer interface {
	// Start outputs a dot (.) every configured interval until Stop is called.
	// Note that Start does not block.
	Start()
	// Stop stops the progress printer, which outputs a newline before exiting.
	// Stop blocks until the progress printer has exited.
	Stop()
}

// NewPrinter creates a new Printer.
func NewPrinter(w io.Writer, interval time.Duration) Printer {
	return printer{
		done:     make(chan bool),
		w:        w,
		interval: interval,
	}
}

type printer struct {
	done     chan bool
	w        io.Writer
	interval time.Duration
}

// Start outputs a dot (.) every configured interval until Stop is called.
// Note that Start does not block.
func (p printer) Start() {
	go func() {
		for {
			tick := time.NewTicker(p.interval)
			select {
			case <-tick.C:
				fmt.Fprint(p.w, ".")
			case <-p.done:
				fmt.Fprintln(p.w)
				p.done <- true
				return
			}
		}
	}()
}

// Stop stops the progress printer, which outputs a newline before exiting.
// Stop blocks until the progress printer has exited.
func (p printer) Stop() {
	p.done <- true
	<-p.done
}
