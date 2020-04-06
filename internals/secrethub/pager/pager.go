package pager

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

const (
	pagerEnvvar            = "$PAGER"
	fallbackPagerLineCount = 100
)

var ErrPagerClosed = errors.New("cannot write to closed terminal pager")
var ErrPagerNotFound = errors.New("no terminal pager available. Please configure a terminal pager by setting the $PAGER environment variable or install \"less\" or \"more\"")

// pager is a writer that is piped to a terminal pager command.
type pager struct {
	writer io.WriteCloser
	cmd    *exec.Cmd
	done   <-chan struct{}
	closed bool
}

// New runs the terminal pager configured in the OS environment
// and returns a writer that is piped to the standard input of the pager command.
func New(outputWriter io.Writer) (io.WriteCloser, error) {
	pagerCommand, err := pagerCommand()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(pagerCommand)

	writer, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stdout = outputWriter
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	done := make(chan struct{}, 1)
	go func() {
		_ = cmd.Wait()
		done <- struct{}{}
	}()
	return &pager{writer: writer, cmd: cmd, done: done}, nil
}

// Write pipes the data to the terminal pager.
// It returns errPagerClosed if the terminal pager has been closed.
func (p *pager) Write(data []byte) (n int, err error) {
	if p.isClosed() {
		return 0, ErrPagerClosed
	}
	return p.writer.Write(data)
}

// Close closes the writer to the terminal pager and waits for the terminal pager to close.
func (p *pager) Close() error {
	err := p.writer.Close()
	if err != nil {
		return err
	}
	if p.closed {
		return nil
	}
	err = p.cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		err = p.cmd.Process.Kill()
		if err != nil {
			return err
		}
	}
	<-p.done
	return nil
}

// isClosed checks if the terminal pager process has been stopped.
func (p *pager) isClosed() bool {
	if p.closed {
		return true
	}
	select {
	case <-p.done:
		p.closed = true
		return true
	default:
		return false
	}
}

// pagerCommand returns the name of the terminal pager configured in the OS environment ($PAGER).
// If no pager is configured it falls back to "less" than "more", returning an error if neither are available.
func pagerCommand() (string, error) {
	if pager, err := exec.LookPath(os.ExpandEnv(pagerEnvvar)); err == nil {
		return pager, nil
	}

	if pager, err := exec.LookPath("less"); err == nil {
		return pager, nil
	}

	if pager, err := exec.LookPath("more"); err == nil {
		return pager, nil
	}

	return "", ErrPagerNotFound
}

// newFallbackPaginatedWriter returns a pager that closes after outputting a fixed number of lines without pagination
// and returns errPagerNotFound on the last (or any subsequent) write.
func NewFallbackPager(w io.WriteCloser) io.WriteCloser {
	return &fallbackPager{
		linesLeft: fallbackPagerLineCount,
		writer:    w,
	}
}

type fallbackPager struct {
	writer    io.WriteCloser
	linesLeft int
}

func (p *fallbackPager) Write(data []byte) (int, error) {
	if p.linesLeft == 0 {
		return 0, ErrPagerNotFound
	}

	lines := bytes.Count(data, []byte{'\n'})
	if lines > p.linesLeft {
		data = bytes.Join(bytes.Split(data, []byte{'\n'})[:p.linesLeft], []byte{'\n'})
		data = append(data, '\n')
	}
	p.linesLeft -= bytes.Count(data, []byte{'\n'})
	n, err := p.writer.Write(data)
	if p.linesLeft == 0 {
		err = ErrPagerNotFound
	}
	return n, err
}

func (p *fallbackPager) Close() error {
	return p.writer.Close()
}
