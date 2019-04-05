package jsonschema

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

var _ RefResolver = &httpRefResolver{}

// httpRefResolver is a RefResolver that uses a standard HTTP GET to fetch
// references, assuming them to be accessible URLs. Resolved schemas are cached
// in the package-scoped DefaultSchemaPool.
type httpRefResolver struct {
	client *http.Client
	cache  Definitions
}

func (h *httpRefResolver) Resolve(ref string) (*Schema, error) {
	refs := h.cache
	if resolved, ok := refs[ref]; ok {
		return resolved, nil
	}

	// NOTE: Parsing & getting the URL returns "not found" on error because this
	// may be called with references which are not remote.
	// See the "Recursive references between schemas" test.
	u, err := url.Parse(ref)
	if err != nil {
		return nil, nil
	}

	res, err := h.client.Get(u.String())
	if err != nil {
		return nil, nil
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	s := &RootSchema{}
	if err := json.NewDecoder(res.Body).Decode(s); err != nil {
		return nil, err
	}
	fmt.Println("GOT IT")
	refs[ref] = &s.Schema
	return &s.Schema, nil
}
