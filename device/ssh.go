package device

import (
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func SimpleConnectionFromSSH(host string, port uint16, config *ssh.ClientConfig) (DeviceConnection, func(), error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, nil, err
	}

	doneCh := make(chan struct{})
	done := func() { close(doneCh) }

	closeWhenDone := func(closer io.Closer) {
		go func() {
			<-doneCh
			closer.Close()
		}()
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to dial")
		done()
		return nil, nil, err
	}
	defer closeWhenDone(client)

	session, err := client.NewSession()
	if err != nil {
		log.Error().Err(err).Msg("failed to create session")
		done()
		return nil, nil, err
	}
	defer closeWhenDone(session)

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // Enable echoing
		ssh.TTY_OP_ISPEED: 14400, // Set input speed to 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // Set output speed to 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 100, 250, modes); err != nil {
		log.Error().Err(err).Msg("failed to request pty")
		done()
		return nil, nil, err
	}

	// Get stdin pipe
	pipeStdin, err := session.StdinPipe()
	if err != nil {
		log.Error().Err(err).Msg("failed to get stdin")
		done()
		return nil, nil, err
	}

	// Get stdout pipe
	pipeStdout, err := session.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("failed to get stdout")
		done()
		return nil, nil, err
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		log.Error().Err(err).Msg("failed to start shell")
		done()
		return nil, nil, err
	}

	conn := NewDeviceConnection(pipeStdin, pipeStdout)
	return conn, done, err
}
