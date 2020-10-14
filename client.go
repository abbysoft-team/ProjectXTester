package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"net"
	"projectx-tester/common"
	rpc "projectx-tester/rpc/generated"
	"time"
)

type Client struct {
	socket        *net.UDPConn
	listenAddress *net.UDPAddr
	serverAddress *net.UDPAddr
	logger        *log.Entry
}

type ClientConfig struct {
	ListenAddress string
	ServerAddress string
}

func NewClient(config ClientConfig) (*Client, error) {
	listenAddress, err := net.ResolveUDPAddr("udp", config.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen address: %w", err)
	}

	serverAddress, err := net.ResolveUDPAddr("udp", config.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server address: %w", err)
	}

	conn, err := net.DialUDP("udp", listenAddress, serverAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create client socket: %w", err)
	}

	logger := log.WithField("module", "Client")

	return &Client{
		socket:        conn,
		listenAddress: listenAddress,
		serverAddress: serverAddress,
		logger:        logger,
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

	go c.readResponses()
	for {
		request := generateRandomRequest()

		if err, _ := common.WriteResponse(&request, nil, c.socket); err != nil {
			c.logger.WithError(err).Error("Failed to write request")
		}

		c.logger.Printf("%T sent to the server", request.Data)
		time.Sleep(5 * time.Second)
	}
}

func (c *Client) readResponses() {
	var buffer [1024]byte
	for {
		bytesRead, err := c.socket.Read(buffer[0:])
		if err != nil {
			c.logger.WithError(err).Error("Failed to read response from the server")
			continue
		}

		var response rpc.Response
		if err := proto.Unmarshal(buffer[0:bytesRead], &response); err != nil {
			c.logger.WithError(err).Error("Failed to deserialize server response")
			continue
		}

		if response.GetMultipartResponse() != nil {
			c.logger.
				WithField("parts", response.GetMultipartResponse().Parts).
				Infof("Server respond with multipart response")

			actualResponse, err := c.readMultipartResponse(int(response.GetMultipartResponse().Parts))
			if err != nil {
				c.logger.WithError(err).Error("Failed to read multipart response")
				continue
			}

			response.Data = actualResponse.Data
		}

		c.logger.
			WithField("response", response.Data).
			Infof("Server respond with %d bytes", bytesRead)
	}
}

func (c *Client) readMultipartResponse(parts int) (*rpc.Response, error) {
	var buffer [common.MaxPacketSize]byte
	var resultBuffer []byte

	for parts > 0 {
		bytesRead, err := c.socket.Read(buffer[0:])
		if err != nil {
			return nil, fmt.Errorf("failed to read response from the server")
		}

		resultBuffer = append(resultBuffer, buffer[0:bytesRead]...)
		c.logger.WithField("length", bytesRead).Debugf("Read response part (%d left)", parts)

		parts--
	}

	c.logger.Debugf("Unmarshal part (%d bytes)", len(resultBuffer))
	var response rpc.Response
	if err := proto.Unmarshal(resultBuffer, &response); err != nil {
		return nil, fmt.Errorf("failed to serialize response: %v", err)
	}

	return &response, nil
}
