package parse

import (
	"strings"

	"github.com/daaku/go.urlbuild"
)

// Specify relations or pointers to include.
func OptionInclude(include []string) urlbuild.Augment {
	if len(include) == 0 {
		return urlbuild.Nil()
	}
	return urlbuild.String("include", strings.Join(include, ","))
}

// Specify the ordering.
func OptionOrder(order string) urlbuild.Augment {
	if order == "" {
		return urlbuild.Nil()
	}
	return urlbuild.String("order", order)
}

// Specify a limit. Note, 0 values are also sent.
func OptionLimit(limit uint64) urlbuild.Augment {
	return urlbuild.Uint64("limit", limit)
}

var optionCount = urlbuild.String("count", "1")

// Specify that total count should be returned.
func OptionCount() urlbuild.Augment {
	return optionCount
}

// Specify number of items to skip. Note, 0 values are not sent.
func OptionSkip(skip uint64) urlbuild.Augment {
	if skip == 0 {
		return urlbuild.Nil()
	}
	return urlbuild.Uint64("skip", skip)
}

// Specify keys to fetch.
func OptionKeys(keys []string) urlbuild.Augment {
	if len(keys) == 0 {
		return urlbuild.Nil()
	}
	return urlbuild.String("keys", strings.Join(keys, ","))
}

// Specify a value to be JSON encoded and used as the where option.
func OptionWhere(where interface{}) urlbuild.Augment {
	return urlbuild.JSON("where", where)
}
