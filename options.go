package parse

import (
	"strings"

	"github.com/daaku/go.urlbuild"
)

// Specify relations or pointers to include.
func ParamInclude(include []string) urlbuild.Param {
	if len(include) == 0 {
		return urlbuild.Nil()
	}
	return urlbuild.String("include", strings.Join(include, ","))
}

// Specify the ordering.
func ParamOrder(order string) urlbuild.Param {
	if order == "" {
		return urlbuild.Nil()
	}
	return urlbuild.String("order", order)
}

// Specify a limit. Note, 0 values are also sent.
func ParamLimit(limit uint64) urlbuild.Param {
	return urlbuild.Uint64("limit", limit)
}

var optionCount = urlbuild.String("count", "1")

// Specify that total count should be returned.
func ParamCount() urlbuild.Param {
	return optionCount
}

// Specify number of items to skip. Note, 0 values are not sent.
func ParamSkip(skip uint64) urlbuild.Param {
	if skip == 0 {
		return urlbuild.Nil()
	}
	return urlbuild.Uint64("skip", skip)
}

// Specify keys to fetch.
func ParamKeys(keys []string) urlbuild.Param {
	if len(keys) == 0 {
		return urlbuild.Nil()
	}
	return urlbuild.String("keys", strings.Join(keys, ","))
}

// Specify a value to be JSON encoded and used as the where option.
func ParamWhere(where interface{}) urlbuild.Param {
	return urlbuild.JSON("where", where)
}
