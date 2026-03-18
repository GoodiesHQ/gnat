package device

import (
	"fmt"
	"io"
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
func NewSSHConnection(host string, port uint16, config *ssh.ClientConfig) (Connection, func(), error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s:%d: %w", host, port, err)
	}

	doneCh := make(chan struct{})
	done := func() { close(doneCh) }

	closeWhenDone := func(c io.Closer) {
		go func() {
			<-doneCh
			c.Close()
		}()
	}
	closeWhenDone(client)

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
		return nil, nil, fmt.Errorf("start shell: %w", err)
	}

	log.Debug().Msgf("SSH session established to %s:%d", host, port)
	return NewConnection(pipeStdin, pipeStdout), done, nil
}
