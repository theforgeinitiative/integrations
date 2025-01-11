package mq

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const doorTopicPrefix = "door/"
const clientID = "forgebot"

type Client struct {
	mqttClient mqtt.Client
}

func NewClient(broker string) (Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetConnectRetry(true)

	c := mqtt.NewClient(opts)
	c.Connect()

	return Client{c}, nil
}

func (c Client) RingDoorbell(door string) error {
	token := c.mqttClient.Publish(doorTopicPrefix+door, 0, false, "ring")
	token.Wait()
	return token.Error()
}
