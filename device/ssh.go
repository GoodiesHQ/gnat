package device

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func AllHostKeyAlgorithms() []string {
	return []string{
		"ssh-rsa",
		"ssh-dss",
		"ecdsa-sha2-nistp256",
		"sk-ecdsa-sha2-nistp256@openssh.com",
		"ecdsa-sha2-nistp384",
		"ecdsa-sha2-nistp521",
		"ssh-ed25519",
		"sk-ssh-ed25519@openssh.com",
		"rsa-sha2-256",
		"rsa-sha2-512",
		"ssh-rsa-cert-v01@openssh.com",
		"ssh-dss-cert-v01@openssh.com",
		"ecdsa-sha2-nistp256-cert-v01@openssh.com",
		"ecdsa-sha2-nistp384-cert-v01@openssh.com",
		"ecdsa-sha2-nistp521-cert-v01@openssh.com",
		"sk-ecdsa-sha2-nistp256-cert-v01@openssh.com",
		"ssh-ed25519-cert-v01@openssh.com",
		"sk-ssh-ed25519-cert-v01@openssh.com",
		"rsa-sha2-256-cert-v01@openssh.com",
		"rsa-sha2-512-cert-v01@openssh.com",
	}
}
func AllCiphers() []string {
	return []string{
		"aes128-ctr",
		"aes192-ctr",
		"aes256-ctr",
		"aes128-gcm@openssh.com",
		"aes256-gcm@openssh.com",
		"chacha20-poly1305@openssh.com",
		"arcfour256",
		"arcfour128",
		"arcfour",
		"aes128-cbc",
		"3des-cbc",
	}
}

func AllKeyExchanges() []string {
	return []string{
		"diffie-hellman-group1-sha1",
		"diffie-hellman-group14-sha1",
		"diffie-hellman-group14-sha256",
		"diffie-hellman-group16-sha512",
		"ecdh-sha2-nistp256",
		"ecdh-sha2-nistp384",
		"ecdh-sha2-nistp521",
		"curve25519-sha256@libssh.org",
		"curve25519-sha256",
		"diffie-hellman-group-exchange-sha1",
		"diffie-hellman-group-exchange-sha256",
	}
}

func UsernamePasswordConfig(username, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback:   ssh.InsecureIgnoreHostKey(),
		HostKeyAlgorithms: AllHostKeyAlgorithms(),
		Config: ssh.Config{
			Ciphers:      AllCiphers(),
			KeyExchanges: AllKeyExchanges(),
		},
		Timeout: 5 * time.Second,
	}
}

// NewSSHConnection dials an SSH session and returns a DeviceConnection plus a done()
// function that closes the session and client when called.
func NewSSHConnection(ctx context.Context, host string, port uint16, config *ssh.ClientConfig) (Connection, func(), error) {
	// target formed from proper host + port joining
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))

	// create an SSH session manually
	dialer := net.Dialer{Timeout: config.Timeout}
	transport, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", target, err)
	}

	connection := &conn{}

	// enforce cleanup exactly once
	doneCh := make(chan struct{})
	var once sync.Once

	done := func() {
		once.Do(func() {
			_ = connection.Stop()
			_ = transport.Close()
			close(doneCh)
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			done()
		case <-doneCh:
		}
	}()

	sshConn, channels, requests, err := ssh.NewClientConn(transport, target, config)
	if err != nil {
		done()
		if ctx.Err() != nil {
			return nil, nil, fmt.Errorf("ssh handshake ctx: %w", ctx.Err())
		}
		return nil, nil, fmt.Errorf("ssh handshake: %w", err)
	}

	// build ssh client from parts
	client := ssh.NewClient(sshConn, channels, requests)
	go func() {
		_ = client.Wait()
		done()
	}()

	closeWhenDone := func(c io.Closer) {
		go func() {
			<-doneCh
			c.Close()
		}()
	}
	closeWhenDone(client)

	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed before ssh session: %w", err)
	}
	session, err := client.NewSession()
	if err != nil {
		done()
		return nil, nil, fmt.Errorf("new session: %w", err)
	}
	closeWhenDone(session)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 100, 250, modes); err != nil {
		done()
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("request pty ctx: %w", err)
		}
		return nil, nil, fmt.Errorf("request pty: %w", err)
	}

	pipeStdin, err := session.StdinPipe()
	if err != nil {
		done()
		return nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}

	pipeStdout, err := session.StdoutPipe()
	if err != nil {
		done()
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := session.Shell(); err != nil {
		done()
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("start shell ctx: %w", err)
		}
		return nil, nil, fmt.Errorf("start shell: %w", err)
	}

	connection.mu.Lock()
	connection.stdin = pipeStdin
	connection.stdout = pipeStdout
	connection.mu.Unlock()

	connection.mu.RLock()
	err = connection.err
	connection.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		done()
		return nil, nil, fmt.Errorf("ssh setup canceled: %w", err)
	}

	if err != nil {
		done()
		return nil, nil, fmt.Errorf("ssh connection failed during setup: %w", err)
	}

	log.Debug().Msgf("ssh session established to %s", target)
	return connection, done, nil
}
