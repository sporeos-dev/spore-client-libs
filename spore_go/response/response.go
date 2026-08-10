package response

import (
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
)

type Response struct {
	h *sporec.Response
}

func New(command string, handle string) *Response {
	r := &Response{h: sporec.ResponseCreate()}
	sporec.ResponseSetCommand(r.h, command)
	sporec.ResponseSetHandle(r.h, handle)
	runtime.SetFinalizer(r, (*Response).destroy)
	return r
}

func FromC(h *sporec.Response) *Response { 
	return &Response{h: h} 
}

func (r *Response) destroy() {
	sporec.ResponseDestroy(r.h)
	r.h = nil
}

func (r *Response) WithArg(key string, value string) *Response {
	sporec.ResponseAddArg(r.h, key, value)
	return r
}

func (r *Response) WithFlag(flag string) *Response {
	sporec.ResponseAddFlag(r.h, flag)
	return r
}

func (r *Response) Command() string {
	return sporec.ResponseGetCommand(r.h)
}

func (r *Response) Handle() string {
	return sporec.ResponseGetHandle(r.h)
}

func (r *Response) Arg(key string) (string, bool) {
	v := sporec.ResponseGetArg(r.h, key)
	if v == "" {
		return "", false
	}
	return v, true
}

func (r *Response) ArgIf(key string, def string) string {
	if v, ok := r.Arg(key); ok {
		return v
	}
	return def
}

func (r *Response) Flag(flag string) bool {
	return sporec.ResponseHasFlag(r.h, flag)
}

func (r *Response) Serialize() string {
	sporec.ResponseSerialize(r.h)
	return sporec.ResponseGetSerialized(r.h)
}

func (r *Response) H() *sporec.Response { return r.h }
