package request

import (
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
)

type Request struct {
	h *sporec.Request
}

func New(command string, handle string) *Request {
	r := &Request{h: sporec.RequestCreate()}
	sporec.RequestSetCommand(r.h, command)
	sporec.RequestSetHandle(r.h, handle)
	runtime.SetFinalizer(r, (*Request).destroy)
	return r
}

func FromC(h *sporec.Request) *Request { 
	return &Request{h: h} 
}

func (r *Request) destroy() {
	sporec.RequestDestroy(r.h)
	r.h = nil
}

func (r *Request) WithArg(key string, value string) *Request {
	sporec.RequestAddArg(r.h, key, value)
	return r
}

func (r *Request) WithFlag(flag string) *Request {
	sporec.RequestAddFlag(r.h, flag)
	return r
}

func (r *Request) Command() string {
	return sporec.RequestGetCommand(r.h)
}

func (r *Request) Handle() string {
	return sporec.RequestGetHandle(r.h)
}

func (r *Request) Arg(key string) (string, bool) {
	v := sporec.RequestGetArg(r.h, key)
	if v == "" {
		return "", false
	}
	return v, true
}

func (r *Request) ArgIf(key string, def string) string {
	if v, ok := r.Arg(key); ok {
		return v
	}
	return def
}	

func (r *Request) Flag(flag string) bool {
	return sporec.RequestHasFlag(r.h, flag)
}

func (r *Request) H() *sporec.Request { return r.h }
