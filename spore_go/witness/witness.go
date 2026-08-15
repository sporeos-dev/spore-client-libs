package witness

import (
	"runtime"

	sporec "github.com/sporeos-dev/spore-client-libs/spore_go/internal/spore_c"
)

type Witness struct {
	h *sporec.Witness
}

func New(body string) *Witness {
	w := &Witness{h: sporec.WitnessCreate()}
	sporec.WitnessSetBody(w.h, body)
	runtime.SetFinalizer(w, (*Witness).destroy)
	return w
}

func (w *Witness) destroy() {
	sporec.WitnessDestroy(w.h)
	w.h = nil
}

func (w *Witness) WithArg(key string, value string) *Witness {
	sporec.WitnessAddArg(w.h, key, value)
	return w
}

func (w *Witness) WithFlag(flag string) *Witness {
	sporec.WitnessAddFlag(w.h, flag)
	return w
}

func (w *Witness) Body() string {
	return sporec.WitnessGetBody(w.h)
}

func (w *Witness) Arg(key string) (string, bool) {
	v := sporec.WitnessGetArg(w.h, key)
	if v == "" {
		return "", false
	}
	return v, true
}

func (w *Witness) ArgIf(key string, def string) string {
	if v, ok := w.Arg(key); ok {
		return v
	}
	return def
}

func (w *Witness) Flag(flag string) bool {
	return sporec.WitnessHasFlag(w.h, flag)
}

func (w *Witness) H() *sporec.Witness { return w.h }

// FromC wraps a raw C witness received via callback — does not own the memory.
func FromC(h *sporec.Witness) *Witness { return &Witness{h: h} }
