package device

import "golang.org/x/crypto/ssh"

func SimpleConnectionFromSSH(host string, port uint16, config ssh.ClientConfig) (DeviceConnection, chan struct{}, error) {

}
