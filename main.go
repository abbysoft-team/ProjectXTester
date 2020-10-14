package main

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func startTestClient(serverAddress string, clientAddress string) {
	client, err := NewClient(ClientConfig{
		ListenAddress: clientAddress,
		ServerAddress: serverAddress,
	})
	if err != nil {
		log.WithError(err).Error("Failed to start client")
		return
	}

	client.Serve()
}

func initLogging() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	initLogging()

	startTestClient("server.abbysoft.org:27015", "0.0.0.0:27016")
	//startTestClient("192.168.1.105:27015", "0.0.0.0:27016")
}