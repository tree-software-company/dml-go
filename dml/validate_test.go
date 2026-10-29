package dml

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type validateCase[T any] struct {
	name string
	data map[string]any
	want []string
}

func runValidateCases[T any](t *testing.T, cases []validateCase[T]) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Decode[T](configWith(tc.data))
			if len(tc.want) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatalf("expected a result")
				}
				return
			}
			if result != nil {
				t.Fatalf("expected a nil result on a validation failure")
			}
			assertViolations(t, err, tc.want...)
		})
	}
}

func assertViolations(t *testing.T, err error, want ...string) {
	t.Helper()

	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("err = %v, want a *ValidationError", err)
	}
	got := make([]string, len(validationErr.Fields))
	for i, field := range validationErr.Fields {
		got[i] = field.describe()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("violations = %q, want %q", got, want)
	}
}

type intBound struct {
	Port int `dml:"port" validate:"min=1,max=65535"`
}

type uintBound struct {
	Workers uint `dml:"workers" validate:"min=2,max=4"`
}

type floatBound struct {
	Ratio float64 `dml:"ratio" validate:"min=0.5,max=1.5"`
}

type stringBound struct {
	Name string `dml:"name" validate:"min=3,max=5"`
}

type listBound struct {
	Tags []string `dml:"tags" validate:"min=1,max=2"`
}

type exactLength struct {
	Code   string            `dml:"code" validate:"len=4"`
	Pair   []int             `dml:"pair" validate:"len=2"`
	Labels map[string]string `dml:"labels" validate:"len=1"`
}

func TestValidateIntBounds(t *testing.T) {
	runValidateCases(t, []validateCase[intBound]{
		{name: "inside", data: map[string]any{"port": 8080}},
		{name: "lower edge", data: map[string]any{"port": 1}},
		{name: "upper edge", data: map[string]any{"port": 65535}},
		{name: "zero is a violation when the key is present", data: map[string]any{"port": 0}, want: []string{"port: min=1"}},
		{name: "above", data: map[string]any{"port": 70000}, want: []string{"port: max=65535"}},
		{name: "negative", data: map[string]any{"port": -5}, want: []string{"port: min=1"}},
		{name: "missing key is skipped without required", data: map[string]any{}},
		{name: "null key is skipped without required", data: map[string]any{"port": nil}},
		{name: "key match ignores case", data: map[string]any{"PORT": 0}, want: []string{"port: min=1"}},
		{name: "float with no fraction", data: map[string]any{"port": 90000.0}, want: []string{"port: max=65535"}},
	})
}

func TestValidateUintBounds(t *testing.T) {
	runValidateCases(t, []validateCase[uintBound]{
		{name: "inside", data: map[string]any{"workers": 3}},
		{name: "lower edge", data: map[string]any{"workers": 2}},
		{name: "upper edge", data: map[string]any{"workers": 4}},
		{name: "below", data: map[string]any{"workers": 1}, want: []string{"workers: min=2"}},
		{name: "above", data: map[string]any{"workers": 5}, want: []string{"workers: max=4"}},
	})
}

func TestValidateFloatBounds(t *testing.T) {
	runValidateCases(t, []validateCase[floatBound]{
		{name: "inside", data: map[string]any{"ratio": 1.0}},
		{name: "lower edge", data: map[string]any{"ratio": 0.5}},
		{name: "upper edge", data: map[string]any{"ratio": 1.5}},
		{name: "below", data: map[string]any{"ratio": 0.4}, want: []string{"ratio: min=0.5"}},
		{name: "above", data: map[string]any{"ratio": 1.6}, want: []string{"ratio: max=1.5"}},
	})
}

func TestValidateStringBoundsCountRunes(t *testing.T) {
	runValidateCases(t, []validateCase[stringBound]{
		{name: "inside", data: map[string]any{"name": "abcd"}},
		{name: "lower edge", data: map[string]any{"name": "abc"}},
		{name: "upper edge", data: map[string]any{"name": "abcde"}},
		{name: "too short", data: map[string]any{"name": "ab"}, want: []string{"name: min=3"}},
		{name: "too long", data: map[string]any{"name": "abcdef"}, want: []string{"name: max=5"}},
		{name: "multibyte runes count once", data: map[string]any{"name": "żółw"}},
		{name: "multibyte too long", data: map[string]any{"name": "żółwiki"}, want: []string{"name: max=5"}},
	})
}

func TestValidateListBounds(t *testing.T) {
	runValidateCases(t, []validateCase[listBound]{
		{name: "inside", data: map[string]any{"tags": []any{"a", "b"}}},
		{name: "empty", data: map[string]any{"tags": []any{}}, want: []string{"tags: min=1"}},
		{name: "too many", data: map[string]any{"tags": []any{"a", "b", "c"}}, want: []string{"tags: max=2"}},
		{name: "missing", data: map[string]any{}},
	})
}

func TestValidateExactLength(t *testing.T) {
	runValidateCases(t, []validateCase[exactLength]{
		{name: "exact", data: map[string]any{"code": "abcd", "pair": []any{1, 2}, "labels": map[string]any{"a": "b"}}},
		{name: "wrong everywhere", data: map[string]any{"code": "ab", "pair": []any{1}, "labels": map[string]any{}}, want: []string{
			"code: len=4",
			"pair: len=2",
			"labels: len=1",
		}},
	})
}

type requiredHolder struct {
	Name  string   `dml:"name" validate:"required"`
	Port  int      `dml:"port" validate:"required"`
	On    bool     `dml:"on" validate:"required"`
	Items []string `dml:"items" validate:"required"`
	Opt   *string  `dml:"opt" validate:"required"`
}

func TestValidateRequired(t *testing.T) {
	runValidateCases(t, []validateCase[requiredHolder]{
		{name: "all present", data: map[string]any{"name": "x", "port": 1, "on": true, "items": []any{"a"}, "opt": "v"}},
		{name: "zero number and false bool count as present", data: map[string]any{"name": "x", "port": 0, "on": false, "items": []any{"a"}, "opt": "v"}},
		{name: "everything missing", data: map[string]any{}, want: []string{
			"name: required",
			"port: required",
			"on: required",
			"items: required",
			"opt: required",
		}},
		{name: "null counts as missing", data: map[string]any{"name": nil, "port": 1, "on": true, "items": []any{"a"}, "opt": "v"}, want: []string{"name: required"}},
		{name: "empty string and empty list fail", data: map[string]any{"name": "", "port": 1, "on": true, "items": []any{}, "opt": "v"}, want: []string{
			"name: required",
			"items: required",
		}},
		{name: "pointer to an empty string fails", data: map[string]any{"name": "x", "port": 1, "on": true, "items": []any{"a"}, "opt": ""}, want: []string{"opt: required"}},
	})
}

type ruledDatabase struct {
	Host string `dml:"host" validate:"required"`
	Port int    `dml:"port" validate:"min=1"`
}

type sectionsHolder struct {
	Database ruledDatabase  `dml:"database"`
	Cache    *ruledDatabase `dml:"cache"`
}

type requiredSection struct {
	Database ruledDatabase `dml:"database" validate:"required"`
}

func TestValidateSections(t *testing.T) {
	t.Run("struct section is mandatory, pointer section is optional", func(t *testing.T) {
		runValidateCases(t, []validateCase[sectionsHolder]{
			{name: "both absent", data: map[string]any{}, want: []string{"database.host: required"}},
			{name: "database only", data: map[string]any{"database": map[string]any{"host": "h"}}},
			{name: "present pointer section is validated", data: map[string]any{
				"database": map[string]any{"host": "h"},
				"cache":    map[string]any{"port": 0},
			}, want: []string{"cache.host: required", "cache.port: min=1"}},
			{name: "null pointer section is skipped", data: map[string]any{
				"database": map[string]any{"host": "h"},
				"cache":    nil,
			}},
		})
	})

	t.Run("required on a section reports only the section", func(t *testing.T) {
		runValidateCases(t, []validateCase[requiredSection]{
			{name: "absent", data: map[string]any{}, want: []string{"database: required"}},
			{name: "present", data: map[string]any{"database": map[string]any{"host": "h"}}},
		})
	})
}

type oneOfHolder struct {
	Mode  string `dml:"mode" validate:"oneof=dev staging prod"`
	Level int    `dml:"level" validate:"oneof=1 2 3"`
	Shard uint   `dml:"shard" validate:"oneof=0 8"`
}

func TestValidateOneOf(t *testing.T) {
	runValidateCases(t, []validateCase[oneOfHolder]{
		{name: "all allowed", data: map[string]any{"mode": "staging", "level": 3, "shard": 8}},
		{name: "zero is allowed when listed", data: map[string]any{"shard": 0}},
		{name: "nothing set", data: map[string]any{}},
		{name: "all rejected", data: map[string]any{"mode": "test", "level": 4, "shard": 7}, want: []string{
			"mode: oneof=dev staging prod",
			"level: oneof=1 2 3",
			"shard: oneof=0 8",
		}},
		{name: "matching is case sensitive", data: map[string]any{"mode": "DEV"}, want: []string{"mode: oneof=dev staging prod"}},
	})
}

type diveHolder struct {
	Ports    []int             `dml:"ports" validate:"min=1,dive,min=1,max=65535"`
	Labels   map[string]string `dml:"labels" validate:"dive,required,max=4"`
	Matrix   [][]int           `dml:"matrix" validate:"dive,min=1,dive,min=0"`
	PtrPorts []*int            `dml:"ptrPorts" validate:"dive,min=1"`
}

func TestValidateDive(t *testing.T) {
	runValidateCases(t, []validateCase[diveHolder]{
		{name: "valid", data: map[string]any{
			"ports":    []any{80, 443},
			"labels":   map[string]any{"env": "dev"},
			"matrix":   []any{[]any{1, 2}, []any{0}},
			"ptrPorts": []any{1, 2},
		}},
		{name: "element paths", data: map[string]any{"ports": []any{80, 70000, 0}}, want: []string{
			"ports[1]: max=65535",
			"ports[2]: min=1",
		}},
		{name: "container rule runs before element rules", data: map[string]any{"ports": []any{}}, want: []string{"ports: min=1"}},
		{name: "map values use sorted keys", data: map[string]any{"labels": map[string]any{"tier": "toolong", "env": ""}}, want: []string{
			"labels.env: required",
			"labels.tier: max=4",
		}},
		{name: "nested dive", data: map[string]any{"matrix": []any{[]any{1, -1}, []any{}}}, want: []string{
			"matrix[0][1]: min=0",
			"matrix[1]: min=1",
		}},
		{name: "pointer elements", data: map[string]any{"ptrPorts": []any{1, 0}}, want: []string{"ptrPorts[1]: min=1"}},
		{name: "null element only triggers required", data: map[string]any{"labels": map[string]any{"env": nil}}, want: []string{"labels.env: required"}},
	})
}

type serverItem struct {
	Name string `dml:"name" validate:"required"`
	Port int    `dml:"port" validate:"min=1"`
}

type fleetHolder struct {
	Servers []serverItem          `dml:"servers"`
	ByName  map[string]serverItem `dml:"byName"`
	Primary *serverItem           `dml:"primary"`
}

func TestValidateNestedCollections(t *testing.T) {
	runValidateCases(t, []validateCase[fleetHolder]{
		{name: "valid", data: map[string]any{
			"servers": []any{map[string]any{"name": "a", "port": 1}},
			"byName":  map[string]any{"x": map[string]any{"name": "x", "port": 2}},
			"primary": map[string]any{"name": "p", "port": 3},
		}},
		{name: "slice elements get index paths", data: map[string]any{
			"servers": []any{
				map[string]any{"name": "a", "port": 1},
				map[string]any{"name": "b", "port": 0},
				map[string]any{"port": 5},
			},
		}, want: []string{"servers[1].port: min=1", "servers[2].name: required"}},
		{name: "map values get key paths", data: map[string]any{
			"byName": map[string]any{"x": map[string]any{"name": "x", "port": 0}},
		}, want: []string{"byName.x.port: min=1"}},
		{name: "pointer to struct", data: map[string]any{
			"primary": map[string]any{"name": "p", "port": 0},
		}, want: []string{"primary.port: min=1"}},
	})
}

type untaggedHolder struct {
	Port int    `dml:"port"`
	Name string `dml:"name"`
}

func TestValidateIgnoresStructsWithoutRules(t *testing.T) {
	result, err := Decode[untaggedHolder](configWith(map[string]any{"port": -1, "name": ""}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Port != -1 {
		t.Fatalf("port = %d, want -1", result.Port)
	}
}

func TestValidateMapTargetsAreNotWalked(t *testing.T) {
	if _, err := Decode[map[string]any](configWith(map[string]any{"port": -1})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePointerTarget(t *testing.T) {
	result, err := Decode[*intBound](configWith(map[string]any{"port": 0}))
	if result != nil {
		t.Fatalf("expected a nil result")
	}
	assertViolations(t, err, "port: min=1")
}

func TestValidateReportsEveryViolation(t *testing.T) {
	type config struct {
		Name string `dml:"name" validate:"required,min=3"`
		Port int    `dml:"port" validate:"min=1"`
		Mode string `dml:"mode" validate:"oneof=dev prod"`
	}

	_, err := Decode[config](configWith(map[string]any{"name": "ab", "port": 0, "mode": "x"}))
	assertViolations(t, err, "name: min=3", "port: min=1", "mode: oneof=dev prod")
}

func TestValidateCapsCollectedViolations(t *testing.T) {
	type config struct {
		Ports []int `dml:"ports" validate:"dive,min=1"`
	}

	zeros := make([]any, maxViolations*2)
	for i := range zeros {
		zeros[i] = 0
	}

	_, err := Decode[config](configWith(map[string]any{"ports": zeros}))
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("err = %v, want a *ValidationError", err)
	}
	if len(validationErr.Fields) != maxViolations {
		t.Fatalf("collected %d violations, want %d", len(validationErr.Fields), maxViolations)
	}
}

func TestValidationErrorShape(t *testing.T) {
	_, err := Decode[intBound](configWith(map[string]any{"port": 0}))

	var fieldErr *FieldError
	if !errors.As(err, &fieldErr) {
		t.Fatalf("err = %v, want a *FieldError inside", err)
	}
	if fieldErr.Path != "port" || fieldErr.Rule != "min" || fieldErr.Param != "1" {
		t.Fatalf("field error = %+v", fieldErr)
	}
	if !errors.Is(fieldErr, ErrValidation) {
		t.Fatalf("a FieldError must wrap ErrValidation")
	}
	if want := "dml: validate port: min=1"; fieldErr.Error() != want {
		t.Fatalf("FieldError.Error() = %q, want %q", fieldErr.Error(), want)
	}
	if want := "dml: validation failed (1): port: min=1"; err.Error() != want {
		t.Fatalf("ValidationError.Error() = %q, want %q", err.Error(), want)
	}
}

func TestValidationErrorTruncatesLongReports(t *testing.T) {
	fields := make([]*FieldError, maxReportedViolations+2)
	for i := range fields {
		fields[i] = newFieldError("key", ruleRequired, "")
	}

	message := (&ValidationError{Fields: fields}).Error()
	if !strings.HasPrefix(message, "dml: validation failed (12): ") {
		t.Fatalf("message = %q", message)
	}
	if !strings.HasSuffix(message, "; and 2 more") {
		t.Fatalf("message = %q", message)
	}
	if got := strings.Count(message, "key: required"); got != maxReportedViolations {
		t.Fatalf("message lists %d violations, want %d", got, maxReportedViolations)
	}
}

func TestFieldErrorPathIsRooted(t *testing.T) {
	cases := map[string]string{
		"":         "root",
		"[1].port": "root[1].port",
		"a.b":      "a.b",
	}
	for path, want := range cases {
		if got := newFieldError(path, ruleMin, "1").Path; got != want {
			t.Errorf("path %q = %q, want %q", path, got, want)
		}
	}
}

type unknownRule struct {
	V int `dml:"v" validate:"bogus"`
}

type badParam struct {
	V int `dml:"v" validate:"min=abc"`
}

type minOnBool struct {
	V bool `dml:"v" validate:"min=1"`
}

type diveOnString struct {
	V string `dml:"v" validate:"dive"`
}

type emptyRule struct {
	V int `dml:"v" validate:"min=1,,max=2"`
}

type diveWithParam struct {
	V []int `dml:"v" validate:"dive=1"`
}

type oneOfWords struct {
	V int `dml:"v" validate:"oneof=a b"`
}

type oneOfNothing struct {
	V string `dml:"v" validate:"oneof"`
}

type negativeLength struct {
	V string `dml:"v" validate:"len=-1"`
}

type requiredWithParam struct {
	V string `dml:"v" validate:"required=yes"`
}

type outOfTypeBound struct {
	V uint `dml:"v" validate:"min=-1"`
}

type nestedBadRule struct {
	Inner unknownRule `dml:"inner"`
}

func TestValidateRejectsMalformedTags(t *testing.T) {
	present := map[string]any{"v": 1}
	absent := map[string]any{}

	t.Run("unknown rule", func(t *testing.T) {
		for name, data := range map[string]map[string]any{"present": present, "absent": absent} {
			t.Run(name, func(t *testing.T) {
				result, err := Decode[unknownRule](configWith(data))
				if result != nil {
					t.Fatalf("expected a nil result")
				}
				assertBindError(t, err, ErrInvalidRule, "v")
			})
		}
	})

	cases := map[string]func(map[string]any) (any, error){
		"non numeric bound":         func(d map[string]any) (any, error) { return Decode[badParam](configWith(d)) },
		"min on a bool":             func(d map[string]any) (any, error) { return Decode[minOnBool](configWith(d)) },
		"dive on a string":          func(d map[string]any) (any, error) { return Decode[diveOnString](configWith(d)) },
		"empty rule":                func(d map[string]any) (any, error) { return Decode[emptyRule](configWith(d)) },
		"dive with a parameter":     func(d map[string]any) (any, error) { return Decode[diveWithParam](configWith(d)) },
		"oneof with words on int":   func(d map[string]any) (any, error) { return Decode[oneOfWords](configWith(d)) },
		"oneof without values":      func(d map[string]any) (any, error) { return Decode[oneOfNothing](configWith(d)) },
		"negative length":           func(d map[string]any) (any, error) { return Decode[negativeLength](configWith(d)) },
		"required with a parameter": func(d map[string]any) (any, error) { return Decode[requiredWithParam](configWith(d)) },
		"negative bound on a uint":  func(d map[string]any) (any, error) { return Decode[outOfTypeBound](configWith(d)) },
	}
	for name, decode := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := decode(absent)
			assertBindError(t, err, ErrInvalidRule, "v")
		})
	}

	t.Run("nested path", func(t *testing.T) {
		_, err := Decode[nestedBadRule](configWith(map[string]any{"inner": map[string]any{"v": 1}}))
		assertBindError(t, err, ErrInvalidRule, "inner.v")
	})
}

func TestValidateErrorsDoNotLeakValues(t *testing.T) {
	const secret = "hunter2-super-secret"

	type config struct {
		Token string   `dml:"token" validate:"min=100"`
		Mode  string   `dml:"mode" validate:"oneof=dev prod"`
		Keys  []string `dml:"keys" validate:"dive,max=3"`
	}

	_, err := Decode[config](configWith(map[string]any{
		"token": secret,
		"mode":  secret,
		"keys":  []any{secret},
	}))
	if err == nil {
		t.Fatalf("expected a validation error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaks a value: %v", err)
	}
}

type textConfig struct {
	Name   string `dml:"name" validate:"required,min=3"`
	Server struct {
		Port int `dml:"port" validate:"min=1,max=65535"`
	} `dml:"server"`
}

func TestUnmarshalValidatesParsedText(t *testing.T) {
	good, err := Unmarshal[textConfig]([]byte("string name = \"api\";\nint server.port = 8080;\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if good.Server.Port != 8080 {
		t.Fatalf("port = %d, want 8080", good.Server.Port)
	}

	bad, err := Unmarshal[textConfig]([]byte("string name = \"ab\";\nint server.port = 70000;\n"))
	if bad != nil {
		t.Fatalf("expected a nil result")
	}
	assertViolations(t, err, "name: min=3", "server.port: max=65535")
}

func TestUnmarshalFileValidatesParsedText(t *testing.T) {
	path := writeTempDML(t, "int server.port = 0;\n")

	result, err := UnmarshalFile[textConfig](path)
	if result != nil {
		t.Fatalf("expected a nil result")
	}
	assertViolations(t, err, "name: required", "server.port: min=1")
}

func TestValidateWorksWithBindRequired(t *testing.T) {
	type config struct {
		Port int `dml:"port,required" validate:"min=1"`
	}

	_, err := Decode[config](configWith(map[string]any{}))
	assertBindError(t, err, ErrMissingRequired, "port")

	_, err = Decode[config](configWith(map[string]any{"port": 0}))
	assertViolations(t, err, "port: min=1")
}
