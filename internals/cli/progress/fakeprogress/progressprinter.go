//go:build !production

// Package fakeprogress provides an implementation of the progress.Printer interface
// to be used in tests.
package fakeprogress

// Printer is a mock of the Printer interface.
type Printer struct {
	Started int
	Stopped int
}

// Start adds 1 to the Started field.
func (p *Printer) Start() {
	p.Started++
}

// Stop adds 1 to the Stopped field.
func (p *Printer) Stop() {
	p.Stopped++
}
