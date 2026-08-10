package callback

import (
	"github.com/sporeos-dev/spore-client-libs/spore_go/publish"
	"github.com/sporeos-dev/spore-client-libs/spore_go/request"
	"github.com/sporeos-dev/spore-client-libs/spore_go/response"
	"github.com/sporeos-dev/spore-client-libs/spore_go/witness"
)

type Request func(request.Request)
type Response func(response.Response)
type Publish func(publish.Publish)
type Witness func(witness.Witness)
