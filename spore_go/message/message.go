package message

import spore "github.com/sporeos-dev/spore-client-libs/spore_go"

type Message interface {
	Send(spore.Client) error
}

