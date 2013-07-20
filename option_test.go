package parse_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/daaku/go.parse"
	"github.com/daaku/go.urlbuild"
)

func TestParamInclude(t *testing.T) {
	t.Parallel()
	const k = "include"
	const v = "a,b"
	expected := url.Values{k: []string{v}}
	actual, err := urlbuild.MakeValues([]urlbuild.Param{
		parse.ParamInclude([]string{"a", "b"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}
