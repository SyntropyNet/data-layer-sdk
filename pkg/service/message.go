package service

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/syntropynet/data-layer-sdk/pkg/options"
)

type Message interface {
	Ack(opts ...nats.AckOpt) error
	AckSync(opts ...nats.AckOpt) error
	Equal(msg Message) bool
	InProgress(opts ...nats.AckOpt) error
	Metadata() (*nats.MsgMetadata, error)
	Nak(opts ...nats.AckOpt) error
	NakWithDelay(delay time.Duration, opts ...nats.AckOpt) error
	Respond(any) error
	Term(opts ...nats.AckOpt) error

	Message() *nats.Msg
	Subject() string
	Reply() string
	Data() []byte
	Header() nats.Header
	QueueName() string
}

type MessageHandler func(msg Message)

type natsMessage struct {
	*nats.Msg
	codec options.Codec
	make  func([]byte, string) (*nats.Msg, error)
}

func wrapMessage(codec options.Codec, maker func([]byte, string) (*nats.Msg, error), msg *nats.Msg) *natsMessage {
	return &natsMessage{Msg: msg, codec: codec, make: maker}
}

func (m natsMessage) Equal(msg Message) bool {
	return m.Msg.Equal(msg.Message())
}

func (m natsMessage) Message() *nats.Msg {
	return m.Msg
}

func (m natsMessage) Respond(msg any) error {
	payload, err := m.codec.Encode(nil, msg)
	if err != nil {
		return err
	}

	nmsg, err := m.make(payload, m.Msg.Reply)
	if err != nil {
		return err
	}

	return m.RespondMsg(nmsg)
}

func (m natsMessage) Subject() string {
	return m.Msg.Subject
}

func (m natsMessage) Reply() string {
	return m.Msg.Reply
}

func (m natsMessage) Data() []byte {
	return m.Msg.Data
}

func (m natsMessage) Header() nats.Header {
	return m.Msg.Header
}

func (m natsMessage) QueueName() string {
	return m.Msg.Sub.Queue
}
