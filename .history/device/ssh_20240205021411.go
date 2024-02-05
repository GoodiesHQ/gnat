package device

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func SimpleConnectionFromSSH(host string, port uint16, config ssh.ClientConfig) (DeviceConnection, chan struct{}, error) {
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", hostname), config)
	if err != nil {
		log.Error().Err(err).Msg("failed to dial")
		return
	}
	defer client.Close()
}
