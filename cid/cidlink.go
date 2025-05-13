package cid

import (
	"encoding/json"
	"fmt"
)

// NOTE: unsure how i want to represent this, might change it

type CidLink struct {
	Bytes []byte
}

type jsonLink struct {
	Link string `json:"$link"`
}

func (ll CidLink) MarshalJSON() ([]byte, error) {
	jl := jsonLink{
		Link: string(ll.Bytes),
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
	*ll = CidLink{Bytes: []byte(c.String())}
	return nil
}
