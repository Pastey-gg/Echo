package queue

import (
	"context"
	"errors"

	"github.com/EvieePy/Echo/models"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

type RMQ struct {
	Publisher *rmq.Publisher
	Env       *rmq.Environment
}

func NewRMQ(config *models.Config) (*RMQ, error) {
	if config.MessageQueue.DSN == "" {
		return nil, errors.New("message_queue config is missing 'dsn'.")
	}

	if config.MessageQueue.Name == "" {
		return nil, errors.New("message_queue config is missing 'name'.")
	}

	addr := config.MessageQueue.DSN
	name := config.MessageQueue.Name

	ctx := context.Background()
	env := rmq.NewEnvironment(addr, nil)

	conn, err := env.NewConnection(ctx)
	if err != nil {
		panic(err)
	}

	_, err = conn.Management().DeclareQueue(ctx, &rmq.QuorumQueueSpecification{Name: name})
	if err != nil {
		panic(err)
	}

	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: name}, nil)
	if err != nil {
		panic(err)
	}

	return &RMQ{Publisher: publisher, Env: env}, nil
}

func (r *RMQ) PublishPaste(pasteId string) error {
	ctx := context.Background()
	err := r.Publisher.PublishAsync(ctx, rmq.NewMessage([]byte(pasteId)), nil)

	return err
}
