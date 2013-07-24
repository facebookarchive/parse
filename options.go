package parse

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// Params are defined by the Parse API and include things like limit/skip etc.
type Param interface {
	set(v url.Values) error
}

type paramInclude []string

func (p paramInclude) set(v url.Values) error {
	if len(p) != 0 {
		v.Add("include", strings.Join(p, ","))
	}
	return nil
}

// Specify relations or pointers to include.
func ParamInclude(include []string) Param {
	return paramInclude(include)
}

type paramOrder string

func (p paramOrder) set(v url.Values) error {
	if p != "" {
		v.Add("order", string(p))
	}
	return nil
}

// Specify the ordering.
func ParamOrder(order string) Param {
	return paramOrder(order)
}

type paramLimit uint64

func (p paramLimit) set(v url.Values) error {
	v.Add("limit", strconv.FormatUint(uint64(p), 10))
	return nil
}

// Specify a limit. Note, 0 values are also sent.
func ParamLimit(limit uint64) Param {
	return paramLimit(limit)
}

type paramCount bool

func (p paramCount) set(v url.Values) error {
	if p {
		v.Add("count", "1")
	}
	return nil
}

// Specify that total count should be returned.
func ParamCount(include bool) Param {
	return paramCount(include)
}

type paramSkip uint64

func (p paramSkip) set(v url.Values) error {
	if p != 0 {
		v.Add("skip", strconv.FormatUint(uint64(p), 10))
	}
	return nil
}

// Specify number of items to skip. Note, 0 values are not sent.
func ParamSkip(skip uint64) Param {
	return paramSkip(skip)
}

type paramKeys []string

func (p paramKeys) set(v url.Values) error {
	if len(p) != 0 {
		v.Add("keys", strings.Join(p, ","))
	}
	return nil
}

// Specify keys to fetch.
func ParamKeys(keys []string) Param {
	return paramKeys(keys)
}

type paramWhere struct {
	Value interface{}
}

func (p *paramWhere) set(v url.Values) error {
	b, err := json.Marshal(p.Value)
	if err != nil {
		return err
	}
	v.Add("where", string(b))
	return nil
}

// Specify a value to be JSON encoded and used as the where option.
func ParamWhere(v interface{}) Param {
	return &paramWhere{Value: v}
}

// Build url.Values from the given Params.
func ParamValues(params ...Param) (v url.Values, err error) {
	v = make(url.Values)
	for _, p := range params {
		err = p.set(v)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}
