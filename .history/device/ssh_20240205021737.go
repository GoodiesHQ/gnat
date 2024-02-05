package device

import (
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func SimpleConnectionFromSSH(host string, port uint16, config *ssh.ClientConfig) (DeviceConnection, chan struct{}, error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)

	done := make(chan struct{})
	closeWhenDone := func (closer io.Closer) {
		<-done
		closer.Close()
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to dial")
		return nil, nil, err
	}

	defer func() { client.Close() }
}
