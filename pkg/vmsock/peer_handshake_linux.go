/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vmsock

import (
	"github.com/linuxkit/virtsock/pkg/vsock"
	log "github.com/sirupsen/logrus"
)

func PeerHandshake() {
	l, err := vsock.Listen(vsock.CIDAny, PeerHandshakePort)
	if err != nil {
		log.Fatalf("PeerHandshake listen for incoming vsock: %v", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Errorf("PeerHandshake accepting incoming socket connection: %v", err)
		}
		_, err = conn.Write([]byte(SeedPhrase))
		if err != nil {
			log.Errorf("PeerHandshake writing seed phrase: %v", err)
		}

		conn.Close()
		log.Info("successful handshake with vsock-host")
	}
}
