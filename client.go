package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq4"
	log "github.com/sirupsen/logrus"
	rpc "projectx-tester/rpc/generated"
	"time"
)

type Client struct {
	socket  *zmq.Socket
	context *zmq.Context
	logger  *log.Entry
	config  ClientConfig
}

type ClientConfig struct {
	ServerEndpoint string
}

func NewClient(config ClientConfig) (*Client, error) {
	context, err := zmq.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq context: %w", err)
	}

	socket, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq socket: %w", err)
	}

	if err := socket.Connect(config.ServerEndpoint); err != nil {
		return nil, fmt.Errorf("failed to connect to the server: %w", err)
	}

	logger := log.WithField("module", "Client")

	return &Client{
		socket:  socket,
		logger:  logger,
		config:  config,
		context: context,
	}, nil
}

func generateRandomRequest() (request rpc.Request) {
	if time.Now().Unix()%2 == 0 {
		request.Data = &rpc.Request_GetMapRequest{
			GetMapRequest: &rpc.GetMapRequest{
				Location: &rpc.Vector3D{
					X: 10,
					Y: 15,
					Z: 20,
				},
			},
		}
	} else {
		request.Data = &rpc.Request_LoginRequest{
			LoginRequest: &rpc.LoginRequest{
				Username: "testCLient",
				Password: "password",
			},
		}
	}

	return
}

func (c *Client) Serve() {
	defer c.socket.Close()

	firstTime := true
	for {
		if !firstTime {
			time.Sleep(5 * time.Second)
		}

		firstTime = false

		request := generateRandomRequest()

		requestBytes, err := proto.Marshal(&request)
		if err != nil {
			c.logger.WithError(err).Error("Failed to marshal request")
			continue
		}

		c.logger.WithFields(log.Fields{
			"server":  c.config.ServerEndpoint,
			"request": fmt.Sprintf("%T", request.Data),
		}).Info("Send request to the server")

		if _, err := c.socket.Send(string(requestBytes), zmq.DONTWAIT); err != nil {
			c.logger.WithError(err).Error("Failed to send request to the server")
			continue
		}

		c.logger.Printf("%T sent to the server", request.Data)

		if err := c.readResponse(); err != nil {
			c.logger.WithError(err).Error("Failed to read response from the server")
		}
	}
}

func (c *Client) readResponse() error {
	bytesRead, err := c.socket.Recv(0)
	if err != nil {
		return fmt.Errorf("failed to read response from the server: %w", err)
	}

	var response rpc.Response
	if err := proto.Unmarshal([]byte(bytesRead), &response); err != nil {
		return fmt.Errorf("failed to unmarshal server response: %w", err)
	}

	if response.GetMultipartResponse() != nil {
		return fmt.Errorf("multipart response received")
	}

	c.logger.
		WithField("response", response.Data).
		Infof("Server respond with %d bytes", len(bytesRead))
	return nil
}
