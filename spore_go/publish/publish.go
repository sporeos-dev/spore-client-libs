package publish

import (
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
)

type Publish struct {
	h *sporec.Publish
}

func New(topic string) *Publish {
	p := &Publish{h: sporec.PublishCreate()}
	sporec.PublishSetTopic(p.h, topic)
	runtime.SetFinalizer(p, (*Publish).destroy)
	return p
}

func FromC(h *sporec.Publish) *Publish { 
	return &Publish{h: h} 
}

func (p *Publish) destroy() {
	sporec.PublishDestroy(p.h)
	p.h = nil
}

func (p *Publish) WithArg(key string, value string) *Publish {
	sporec.PublishAddArg(p.h, key, value)
	return p
}

func (p *Publish) WithFlag(flag string) *Publish {
	sporec.PublishAddFlag(p.h, flag)
	return p
}

func (p *Publish) Topic() string {
	return sporec.PublishGetTopic(p.h)
}

func (p *Publish) Arg(key string) (string, bool) {
	v := sporec.PublishGetArg(p.h, key)
	if v == "" {
		return "", false
	}
	return v, true
}

func (p *Publish) ArgIf(key string, def string) string {
	if v, ok := p.Arg(key); ok {
		return v
	}
	return def
}

func (p *Publish) Flag(flag string) bool {
	return sporec.PublishHasFlag(p.h, flag)
}

func (p *Publish) H() *sporec.Publish { return p.h }

func (p *Publish) Serialize() string {
	sporec.PublishSerialize(p.h)
	return sporec.PublishGetSerialized(p.h)
}

