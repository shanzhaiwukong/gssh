package gssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type (
	writerbuffer struct {
		buffer bytes.Buffer
		mu     sync.Mutex
	}
	// SSH struct
	SSH struct {
		client      *ssh.Client
		session     *ssh.Session
		stdinPipe   io.WriteCloser
		stdoutPipie *writerbuffer
	}
)

func (w *writerbuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// New ssh
func New(ip, user, pwd string, port int) (*SSH, error) {
	ams := []ssh.AuthMethod{ssh.Password(pwd)}
	cfg := ssh.ClientConfig{User: user, Auth: ams, HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }}
	if cli, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), &cfg); err != nil {
		return nil, err
	} else {
		return &SSH{client: cli}, nil
	}
}

// NewSession create session
func (s *SSH) NewSession(rows, cols int) error {
	if s.client == nil {
		return errors.New("client is nil")
	}
	sess, err := s.client.NewSession()
	if err != nil {
		return err
	}
	stdip, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	s.stdinPipe = stdip
	comboWrite := new(writerbuffer)
	sess.Stderr = comboWrite
	sess.Stdout = comboWrite
	s.stdoutPipie = comboWrite
	stm := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = sess.RequestPty("xterm", rows, cols, stm)
	if err != nil {
		return err
	}
	err = sess.Shell()
	if err != nil {
		return err
	}
	s.session = sess
	return nil
}

// BindInOutExitChannel  Bind input stream channel、output stream channel、exit channel
func (s *SSH) BindInOutExitChannel(chIn chan []byte, chOut chan []byte, chExit chan bool) {
	go s.recv(chIn, chExit)
	go s.back(chOut, chExit)
	go s.wait(chExit)
}

func (s *SSH) recv(chIn chan []byte, chExit chan bool) {
	defer func() {
		chExit <- true
	}()
	for {
		select {
		case <-chExit:
			return
		case data := <-chIn:
			if _, err := s.stdinPipe.Write(data); err != nil {
				fmt.Println("write ssh data error", err)
			}
		}
	}
}
func (s *SSH) back(chOut chan []byte, chExit chan bool) {
	defer func() {
		chExit <- true
	}()
	tick := time.NewTicker(time.Millisecond * time.Duration(120))
	for {
		select {
		case <-tick.C:
			if s.stdoutPipie.buffer.Len() != 0 {
				chOut <- s.stdoutPipie.buffer.Bytes()
				s.stdoutPipie.buffer.Reset()
			}
		case <-chExit:
			return
		}
	}
}
func (s *SSH) wait(chExit chan bool) {
	if s.session != nil {
		if err := s.session.Wait(); err != nil {
			fmt.Println("ssh wait err", err)
			chExit <- true
		}
	}
}

// Close will fisrt close session then close client
func (s *SSH) Close() (sessionError error, clientError error) {
	if s.session != nil {
		sessionError = s.session.Close()
	} else {
		sessionError = errors.New("session is nil")
	}
	if s.client != nil {
		clientError = s.client.Close()
	} else {
		clientError = errors.New("client is nil")
	}
	return
}

// ChangeSize change window size
func (s *SSH) ChangeSize(rows, cols int) error {
	if s.session != nil {
		return s.session.WindowChange(rows, cols)
	}
	return errors.New("session is nil")
}
