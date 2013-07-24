package parse_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/daaku/go.parse"
)

func TestParamInclude(t *testing.T) {
	t.Parallel()
	const k = "include"
	const v = "a,b"
	expected := url.Values{k: []string{v}}
	actual, err := parse.ParamValues(parse.ParamInclude([]string{"a", "b"}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamIncludeEmpty(t *testing.T) {
	t.Parallel()
	expected := url.Values{}
	actual, err := parse.ParamValues(parse.ParamInclude([]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamOrder(t *testing.T) {
	t.Parallel()
	const k = "order"
	const v = "a"
	expected := url.Values{k: []string{v}}
	actual, err := parse.ParamValues(parse.ParamOrder(v))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamOrderEmpty(t *testing.T) {
	t.Parallel()
	expected := url.Values{}
	actual, err := parse.ParamValues(parse.ParamOrder(""))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamLimit(t *testing.T) {
	t.Parallel()
	const k = "limit"
	const v = "0"
	expected := url.Values{k: []string{v}}
	actual, err := parse.ParamValues(parse.ParamLimit(0))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamCount(t *testing.T) {
	t.Parallel()
	const k = "count"
	const v = "1"
	expected := url.Values{k: []string{v}}
	actual, err := parse.ParamValues(parse.ParamCount(true))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamSkip(t *testing.T) {
	t.Parallel()
	const k = "skip"
	const v = "1"
	expected := url.Values{k: []string{v}}
	actual, err := parse.ParamValues(parse.ParamSkip(1))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamSkipZero(t *testing.T) {
	t.Parallel()
	expected := url.Values{}
	actual, err := parse.ParamValues(parse.ParamSkip(0))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamKeys(t *testing.T) {
	t.Parallel()
	const k = "keys"
	const v = "a,b"
	expected := url.Values{k: []string{v}}
	actual, err := parse.ParamValues(parse.ParamKeys([]string{"a", "b"}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamKeysEmpty(t *testing.T) {
	t.Parallel()
	expected := url.Values{}
	actual, err := parse.ParamValues(parse.ParamKeys([]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamWhere(t *testing.T) {
	t.Parallel()
	const k = "where"
	v := map[string]int{"a": 42}
	expected := url.Values{k: []string{`{"a":42}`}}
	actual, err := parse.ParamValues(parse.ParamWhere(v))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%+v\nactual:\n%+v", expected, actual)
	}
}

func TestParamWhereError(t *testing.T) {
	t.Parallel()
	const k = "where"
	v := map[int]int{}
	_, err := parse.ParamValues(parse.ParamWhere(v))
	if err == nil {
		t.Fatal("was expecting error")
	}
}