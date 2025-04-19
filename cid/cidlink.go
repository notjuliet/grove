package cid

import (
	"encoding/json"
	"fmt"
)

// NOTE: unsure how i want to represent this, might change it

type CidLink string

type jsonLink struct {
	Link string `json:"$link"`
}

func (ll CidLink) MarshalJSON() ([]byte, error) {
	jl := jsonLink{
		Link: string(ll),
	}
	return json.Marshal(jl)
}

func (ll *CidLink) UnmarshalJSON(raw []byte) error {
	var jl jsonLink
	if err := json.Unmarshal(raw, &jl); err != nil {
		return fmt.Errorf("parsing cid-link JSON: %v", err)
	}

	c, err := Parse(jl.Link)
	if err != nil {
		return fmt.Errorf("parsing cid-link CID: %v", err)
	}
	*ll = CidLink(c.String())
	return nil
}
