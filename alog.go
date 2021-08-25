// Package alog provides a simple asynchronous logger that will write to provided io.Writers without blocking calling
// goroutines.
package alog

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Alog is a type that defines a logger. It can be used to write log messages synchronously (via the Write method)
// or asynchronously via the channel returned by the MessageChannel accessor.
type Alog struct {
	dest               io.Writer
	m                  *sync.Mutex
	msgCh              chan string
	errorCh            chan error
	shutdownCh         chan bool
	shutdownCompleteCh chan bool
}

// New creates a new Alog object that writes to the provided io.Writer.
// If nil is provided the output will be directed to os.Stdout.
func New(w io.Writer) *Alog {
	msgCh := make(chan string)
	errorCh := make(chan error)
	shutdownCh := make(chan bool)
	shutdownCompleteCh := make(chan bool)
	if w == nil {
		w = os.Stdout
	}
	return &Alog{
		dest:               w,
		m:                  &sync.Mutex{},
		msgCh:              msgCh,
		errorCh:            errorCh,
		shutdownCh:         shutdownCh,
		shutdownCompleteCh: shutdownCompleteCh,
	}
}

// Start begins the message loop for the asynchronous logger. It should be initiated as a goroutine to prevent
// the caller from being blocked.
func (al Alog) Start() {
	var wg sync.WaitGroup
outerloop:
	for {
		select {
		case <-al.msgCh:
			go func() {
				fmt.Println("write")
				al.write(<-al.msgCh, &wg)
				fmt.Println("written")

			}()
		case <-al.shutdownCh:
			al.shutdown()
			break outerloop
		}
	}
}

func (al Alog) formatMessage(msg string) string {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return fmt.Sprintf("[%v] - %v", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func (al Alog) write(msg string, wg *sync.WaitGroup) {
	formattedMessage := al.formatMessage(msg)

	fmt.Printf("Locking %s \n", formattedMessage)
	al.m.Lock()
	fmt.Printf("locked - formatted message %s \n", formattedMessage)

	_, err := al.dest.Write([]byte(formattedMessage))
	fmt.Printf("formatted message %s \n", formattedMessage)

	al.m.Unlock()

	al.errorCh <- err
	wg.Done()
}

func (al Alog) shutdown() {
	close(al.msgCh)
	al.shutdownCompleteCh <- true
}

// MessageChannel returns a channel that accepts messages that should be written to the log.
func (al Alog) MessageChannel() chan<- string {
	return al.msgCh
}

// ErrorChannel returns a channel that will be populated when an error is raised during a write operation.
// This channel should always be monitored in some way to prevent deadlock goroutines from being generated
// when errors occur.
func (al Alog) ErrorChannel() <-chan error {
	return al.errorCh
}

// Stop shuts down the logger. It will wait for all pending messages to be written and then return.
// The logger will no longer function after this method has been called.
func (al Alog) Stop() {
	al.shutdownCh <- true
	<-al.shutdownCompleteCh
}

// Write synchronously sends the message to the log output
func (al Alog) Write(msg string) (int, error) {
	return al.dest.Write([]byte(al.formatMessage(msg)))
}
