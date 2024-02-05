package device

import (
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func SimpleConnectionFromSSH(host string, port uint16, config *ssh.ClientConfig) (DeviceConnection, chan struct{}, error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)

	doneCh := make(chan struct{}, 0)
	closeWhenDone := func(closer io.Closer) {
		<-doneCh
		closer.Close()
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to dial")
		close(doneCh)
		return nil, nil, err
	}
	defer closeWhenDone(client)

	session, err := client.NewSession()
	if err != nil {
		log.Error().Err(err).Msg("failed to create session")
		close(doneCh)
		return nil, nil, err
	}
	defer closeWhenDone(client)
}
