package response

import (
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
)

type ResponseError struct {
	h *sporec.ResponseError
}

func Error(command string, handle string, code string, what string) *ResponseError {
	re := &ResponseError{h: sporec.ResponseErrorCreate()}
	sporec.ResponseErrorSetCommand(re.h, command)
	sporec.ResponseErrorSetHandle(re.h, handle)
	sporec.ResponseErrorSetCode(re.h, code)
	sporec.ResponseErrorSetWhat(re.h, what)
	runtime.SetFinalizer(re, (*ResponseError).destroy)
	return re
}

func ErrorFromC(h *sporec.ResponseError) *ResponseError { 
	return &ResponseError{h: h} 
}

func (re *ResponseError) destroy() {
	sporec.ResponseErrorDestroy(re.h)
	re.h = nil
}

func (re *ResponseError) WithArg(key string, value string) *ResponseError {
	sporec.ResponseErrorAddArg(re.h, key, value)
	return re
}

func (re *ResponseError) WithFlag(flag string) *ResponseError {
	sporec.ResponseErrorAddFlag(re.h, flag)
	return re
}

func (re *ResponseError) Code() string {
	return sporec.ResponseErrorGetCode(re.h)
}

func (re *ResponseError) What() string {
	return sporec.ResponseErrorGetWhat(re.h)
}

func (re *ResponseError) Command() string {
	return sporec.ResponseErrorGetCommand(re.h)
}

func (re *ResponseError) Handle() string {
	return sporec.ResponseErrorGetHandle(re.h)
}

func (re *ResponseError) H() *sporec.ResponseError { return re.h }

func (re *ResponseError) Arg(key string) (string, bool) {
	v := sporec.ResponseErrorGetArg(re.h, key)
	if v == "" {
		return "", false
	}
	return v, true
}

func (re *ResponseError) ArgIf(key string, def string) string {
	if v, ok := re.Arg(key); ok {
		return v
	}
	return def
}

func (re *ResponseError) Flag(flag string) bool {
	return sporec.ResponseErrorHasFlag(re.h, flag)
}

func (re *ResponseError) Serialize() string {
	sporec.ResponseErrorSerialize(re.h)
	return sporec.ResponseErrorGetSerialized(re.h)
}



