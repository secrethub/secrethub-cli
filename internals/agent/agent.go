package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/mitchellh/go-ps"

	"github.com/secrethub/secrethub-cli/internals/agent/protocol"
	"github.com/secrethub/secrethub-cli/internals/cli"
)

type contextKey int

const (
	ppidKey = contextKey(1)
)

type Server struct {
	dirPath string
	version string
	logger  cli.Logger
}

func New(configDir string, version string, logger cli.Logger) *Server {
	return &Server{
		dirPath: configDir,
		version: version,
		logger:  logger,
	}
}

func (s *Server) Start() error {
	isRunning, pid, err := s.IsRunning()
	if err != nil {
		return err
	}
	if isRunning {
		s.logger.Debugf("agent already running with pid %d", pid)
		return errors.New("already running")
	}

	listener, err := net.Listen("unix", s.socketPath())
	if err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	if err := os.Chmod(s.socketPath(), 0200); err != nil {
		return fmt.Errorf("set socket permission: %v", err)
	}

	server := http.Server{
		Handler: newController(s.version).handler(),
		ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
			ppid, err := getConnPpid(conn)
			if err != nil {
				return ctx
			}
			return context.WithValue(ctx, ppidKey, ppid)
		},
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig

		s.logger.Debugf("stopping listener")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := server.Shutdown(ctx)
		cancel()
		if err != nil {
			fmt.Printf("Could not close server: %s\n", err)
		}

		s.logger.Debugf("deleting pid file")

		err = s.deletePIDFile()
		if err != nil {
			fmt.Printf("Could not delete pid file: %s\n", err)
		}

		s.logger.Debugf("agent stopped")
	}()

	err = s.writePIDFile()
	if err != nil {
		return fmt.Errorf("cannot write pid file: %v", err)
	}
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Restart() error {
	err := s.Kill()
	if err != nil {
		return err
	}

	return s.Start()
}

func (s *Server) IsRunning() (bool, int, error) {
	pidContent, err := ioutil.ReadFile(s.pidFilePath())
	if os.IsNotExist(err) {
		return false, 0, nil
	} else if err != nil {
		return false, 0, fmt.Errorf("cannot read pid file: %v", err)
	}
	pid, err := strconv.Atoi(string(bytes.TrimSpace(pidContent)))
	if err != nil {
		return false, 0, fmt.Errorf("pid file corrupted: %v", err)
	}

	psProc, err := ps.FindProcess(pid)
	if err != nil {
		return false, 0, fmt.Errorf("cannot find agent process: %v", err)
	}
	if psProc == nil {
		return false, 0, nil
	}
	return true, pid, nil
}

func (s *Server) Kill() error {
	running, pid, err := s.IsRunning()
	if err != nil {
		return err
	}
	if !running {
		err := os.Remove(s.socketPath())
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cannot remove socket file: %v", err)
		}
		err = s.deletePIDFile()
		if err != nil {
			return fmt.Errorf("cannot delete pid file: %v", err)
		}
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot find process: %v", err)
	}
	err = proc.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("cannot kill process: %v", err)
	}
	err = s.waitForKilled()
	if err != nil {
		return fmt.Errorf("could not determine process end: %v", err)
	}
	return nil
}

func (s *Server) waitForKilled() error {
	s.logger.Debugf("waiting for agent to be stopped")
	backoffTime := time.Millisecond
	for {
		isRunning, _, err := s.IsRunning()
		if err != nil {
			return err
		}
		if !isRunning {
			return nil
		}

		if backoffTime > 10*time.Second {
			break
		}
		s.logger.Debugf("could not find agent, retrying in %s", backoffTime)

		<-time.After(backoffTime)
		backoffTime *= 2
	}
	return errors.New("timeout")
}

func (s *Server) socketPath() string {
	return filepath.Join(s.dirPath, protocol.SocketName)
}

func (s *Server) pidFilePath() string {
	return filepath.Join(s.dirPath, protocol.PIDFileName)
}

func (s *Server) writePIDFile() error {
	return ioutil.WriteFile(s.pidFilePath(), []byte(strconv.Itoa(os.Getpid())), os.FileMode(0644))
}

func (s *Server) deletePIDFile() error {
	err := os.Remove(s.pidFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func getConnPpid(conn net.Conn) (int, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, errors.New("not a unix socket")
	}
	f, err := unixConn.File()
	if err != nil {
		return 0, fmt.Errorf("cannot get underlying file: %s", err)
	}
	defer f.Close()

	cred, err := syscall.GetsockoptUcred(int(f.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		return 0, fmt.Errorf("cannot get peer credential: %s", err)
	}

	proc, err := ps.FindProcess(int(cred.Pid))
	if err != nil {
		return 0, fmt.Errorf("cannot find process: %s", err)
	}
	return proc.PPid(), nil
}
