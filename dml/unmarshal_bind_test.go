package dml

import (
	"errors"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

type holder[T any] struct {
	V T
}

type bindCase[T any] struct {
	name     string
	data     map[string]any
	want     T
	wantErr  error
	wantPath string
}

func runBindCases[T any](t *testing.T, cases []bindCase[T]) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode[T](configWith(tc.data))
			if tc.wantErr != nil {
				assertBindError(t, err, tc.wantErr, tc.wantPath)
				if got != nil {
					t.Errorf("got %+v, want nil on error", *got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("got %+v, want %+v", *got, tc.want)
			}
		})
	}
}

func expectValue[T any](t *testing.T, source any, want T) {
	t.Helper()

	got, err := Decode[holder[T]](configWith(map[string]any{"V": source}))
	if err != nil {
		t.Fatalf("Decode(%#v): %v", source, err)
	}
	if !reflect.DeepEqual(got.V, want) {
		t.Errorf("Decode(%#v) = %#v, want %#v", source, got.V, want)
	}
}

func expectFailure[T any](t *testing.T, source any, wantErr error) {
	t.Helper()

	got, err := Decode[holder[T]](configWith(map[string]any{"V": source}))
	assertBindError(t, err, wantErr, "V")
	if got != nil {
		t.Errorf("Decode(%#v) = %+v, want nil on error", source, *got)
	}
}

type taggedConfig struct {
	Renamed  string `dml:"original"`
	Port     int
	Skipped  string `dml:"-"`
	private  string
	Required string `dml:"needed,required"`
}

func TestBindStructTags(t *testing.T) {
	runBindCases(t, []bindCase[taggedConfig]{
		{
			name: "tag name wins over the field name",
			data: map[string]any{"original": "from-tag", "Renamed": "ignored", "needed": "y"},
			want: taggedConfig{Renamed: "from-tag", Required: "y"},
		},
		{
			name: "field name is matched without case",
			data: map[string]any{"PORT": 80, "needed": "y"},
			want: taggedConfig{Port: 80, Required: "y"},
		},
		{
			name: "exact key beats a case variant",
			data: map[string]any{"port": 1, "Port": 2, "needed": "y"},
			want: taggedConfig{Port: 2, Required: "y"},
		},
		{
			name: "skipped and unexported fields are never filled",
			data: map[string]any{"Skipped": "x", "-": "x", "private": "x", "needed": "y"},
			want: taggedConfig{Required: "y"},
		},
		{
			name: "missing keys leave zero values and unknown keys are ignored",
			data: map[string]any{"needed": "y", "unknown": 1},
			want: taggedConfig{Required: "y"},
		},
		{
			name:     "required key that is missing",
			data:     map[string]any{"Port": 1},
			wantErr:  ErrMissingRequired,
			wantPath: "needed",
		},
		{
			name:     "required key that is null",
			data:     map[string]any{"needed": nil},
			wantErr:  ErrMissingRequired,
			wantPath: "needed",
		},
	})
}

type portOnly struct {
	Port int
}

func TestBindAmbiguousCaseVariantsAreDeterministic(t *testing.T) {
	data := map[string]any{"port": 1, "PORT": 2, "pOrT": 3}

	for i := 0; i < 20; i++ {
		got, err := Decode[portOnly](configWith(data))
		if err != nil || got.Port != 2 {
			t.Fatalf("run %d: got %+v, err %v, want the first key in sorted order", i, got, err)
		}
	}
}

func TestBindPrivateFieldStaysEmpty(t *testing.T) {
	got, err := Decode[taggedConfig](configWith(map[string]any{"private": "x", "needed": "y"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.private != "" {
		t.Errorf("private = %q, want empty", got.private)
	}
}

func TestBindNilSourceLeavesZeroValue(t *testing.T) {
	expectValue[int](t, nil, 0)
	expectValue[string](t, nil, "")
	expectValue[[]string](t, nil, nil)
	expectValue[*int](t, nil, nil)
	expectValue[map[string]int](t, nil, nil)
}

func TestBindIntegerTargets(t *testing.T) {
	t.Run("signed sources", func(t *testing.T) {
		expectValue(t, 100, int8(100))
		expectValue(t, -100, int16(-100))
		expectValue(t, 100, int32(100))
		expectValue(t, 100, int64(100))
		expectValue(t, int64(8080), 8080)
		expectValue(t, int8(5), int64(5))
	})

	t.Run("unsigned sources", func(t *testing.T) {
		expectValue(t, uint(5), 5)
		expectValue(t, uint8(200), int32(200))
	})

	t.Run("integral floats", func(t *testing.T) {
		expectValue(t, 8080.0, 8080)
		expectValue(t, -3.0, int8(-3))
		expectValue(t, float32(7), int64(7))
	})

	t.Run("overflow", func(t *testing.T) {
		expectFailure[int8](t, 300, ErrOutOfRange)
		expectFailure[int8](t, -129, ErrOutOfRange)
		expectFailure[int16](t, 40000, ErrOutOfRange)
		expectFailure[int32](t, int64(math.MaxInt32)+1, ErrOutOfRange)
		expectFailure[int](t, uint64(math.MaxUint64), ErrOutOfRange)
		expectFailure[int](t, 1e30, ErrOutOfRange)
		expectFailure[int](t, -1e30, ErrOutOfRange)
		expectFailure[int8](t, 300.0, ErrOutOfRange)
	})

	t.Run("fractions and non-finite numbers", func(t *testing.T) {
		expectFailure[int](t, 1.5, ErrTypeMismatch)
		expectFailure[int](t, math.NaN(), ErrTypeMismatch)
		expectFailure[int](t, math.Inf(1), ErrTypeMismatch)
		expectFailure[int](t, math.Inf(-1), ErrTypeMismatch)
	})
}

func TestBindUnsignedTargets(t *testing.T) {
	t.Run("accepted sources", func(t *testing.T) {
		expectValue(t, 5, uint(5))
		expectValue(t, 200, uint8(200))
		expectValue(t, uint64(math.MaxUint64), uint64(math.MaxUint64))
		expectValue(t, 3.0, uint16(3))
	})

	t.Run("out of range", func(t *testing.T) {
		expectFailure[uint](t, -1, ErrOutOfRange)
		expectFailure[uint8](t, 300, ErrOutOfRange)
		expectFailure[uint](t, -1.0, ErrOutOfRange)
		expectFailure[uint64](t, 1e30, ErrOutOfRange)
	})

	t.Run("fractions and non-finite numbers", func(t *testing.T) {
		expectFailure[uint](t, 1.5, ErrTypeMismatch)
		expectFailure[uint](t, math.NaN(), ErrTypeMismatch)
		expectFailure[uint](t, math.Inf(1), ErrTypeMismatch)
	})
}

func TestBindFloatTargets(t *testing.T) {
	t.Run("accepted sources", func(t *testing.T) {
		expectValue(t, 3, 3.0)
		expectValue(t, 3, float32(3))
		expectValue(t, 0.5, float32(0.5))
		expectValue(t, uint(7), 7.0)
		expectValue(t, int64(1)<<53, float64(1<<53))
		expectValue(t, int64(1)<<24, float32(1<<24))
	})

	t.Run("precision and range", func(t *testing.T) {
		expectFailure[float64](t, int64(1)<<53+1, ErrOutOfRange)
		expectFailure[float32](t, int64(1)<<24+1, ErrOutOfRange)
		expectFailure[float64](t, uint64(1)<<53+1, ErrOutOfRange)
		expectFailure[float32](t, 1e300, ErrOutOfRange)
	})

	t.Run("not a number is kept", func(t *testing.T) {
		got, err := Decode[holder[float64]](configWith(map[string]any{"V": math.NaN()}))
		if err != nil {
			t.Fatal(err)
		}
		if !math.IsNaN(got.V) {
			t.Errorf("V = %v, want NaN", got.V)
		}
	})
}

func TestBindDoesNotMixStringsNumbersAndBooleans(t *testing.T) {
	expectFailure[int](t, "8080", ErrTypeMismatch)
	expectFailure[float64](t, "1.5", ErrTypeMismatch)
	expectFailure[string](t, 8080, ErrTypeMismatch)
	expectFailure[bool](t, 1, ErrTypeMismatch)
	expectFailure[bool](t, "true", ErrTypeMismatch)
	expectFailure[int](t, true, ErrTypeMismatch)
	expectFailure[string](t, true, ErrTypeMismatch)
	expectFailure[int](t, []any{1}, ErrTypeMismatch)
	expectFailure[string](t, map[string]any{}, ErrTypeMismatch)
}

func TestBindErrorDoesNotLeakValues(t *testing.T) {
	_, err := Decode[holder[int]](configWith(map[string]any{"V": "zzz-secret"}))
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "zzz-secret") {
		t.Errorf("error leaks the value: %v", err)
	}

	_, err = Decode[holder[int8]](configWith(map[string]any{"V": 987654321}))
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "987654321") {
		t.Errorf("error leaks the number: %v", err)
	}
}

func TestBindErrorMessage(t *testing.T) {
	err := newBindError("server.port", ErrTypeMismatch, "expected int, got string")

	want := "dml: bind server.port: value does not match the field type (expected int, got string)"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, ErrTypeMismatch) {
		t.Error("errors.Is should reach the wrapped sentinel")
	}
	if newBindError("", ErrTooDeep, "").Path != "root" {
		t.Error("an empty path should be reported as root")
	}
	if newBindError("[3]", ErrTooDeep, "").Path != "root[3]" {
		t.Error("an index path at the root should be prefixed with root")
	}
	if got := newBindError("a", ErrTooDeep, "").Error(); got != "dml: bind a: value is nested too deeply" {
		t.Errorf("Error() without detail = %q", got)
	}
}

type pathPort struct {
	Port int `dml:"port"`
}

type pathServer struct {
	Name string `dml:"name"`
}

type pathConfig struct {
	Server  pathPort       `dml:"server"`
	Servers []pathServer   `dml:"servers"`
	Labels  map[string]int `dml:"labels"`
}

func TestBindErrorPaths(t *testing.T) {
	runBindCases(t, []bindCase[pathConfig]{
		{
			name:     "nested struct field",
			data:     map[string]any{"server": map[string]any{"port": "x"}},
			wantErr:  ErrTypeMismatch,
			wantPath: "server.port",
		},
		{
			name: "slice element field",
			data: map[string]any{"servers": []any{
				map[string]any{"name": "a"},
				map[string]any{"name": "b"},
				map[string]any{"name": 5},
			}},
			wantErr:  ErrTypeMismatch,
			wantPath: "servers[2].name",
		},
		{
			name:     "map entry",
			data:     map[string]any{"labels": map[string]any{"env": "x"}},
			wantErr:  ErrTypeMismatch,
			wantPath: "labels.env",
		},
		{
			name:     "struct given a list",
			data:     map[string]any{"server": []any{1}},
			wantErr:  ErrTypeMismatch,
			wantPath: "server",
		},
	})
}

func TestBindErrorPathSanitizesDataKeys(t *testing.T) {
	_, err := Decode[pathConfig](configWith(map[string]any{
		"labels": map[string]any{"a\nb": "x"},
	}))
	assertBindError(t, err, ErrTypeMismatch, `labels.a\nb`)

	long := strings.Repeat("k", 200)
	_, err = Decode[pathConfig](configWith(map[string]any{
		"labels": map[string]any{long: "x"},
	}))
	assertBindError(t, err, ErrTypeMismatch, "labels."+strings.Repeat("k", maxPathSegmentLength))
}

type pointerConfig struct {
	Count  *int
	Server *serverSection
	Deep   **int
}

func TestBindPointers(t *testing.T) {
	t.Run("present values allocate", func(t *testing.T) {
		got, err := Decode[pointerConfig](configWith(map[string]any{
			"Count":  5,
			"Server": map[string]any{"Port": 1},
			"Deep":   7,
		}))
		if err != nil {
			t.Fatal(err)
		}
		if got.Count == nil || *got.Count != 5 {
			t.Errorf("Count = %v, want 5", got.Count)
		}
		if got.Server == nil || got.Server.Port != 1 {
			t.Errorf("Server = %+v, want port 1", got.Server)
		}
		if got.Deep == nil || *got.Deep == nil || **got.Deep != 7 {
			t.Errorf("Deep = %v, want 7", got.Deep)
		}
	})

	t.Run("zero is distinguishable from missing", func(t *testing.T) {
		got, err := Decode[pointerConfig](configWith(map[string]any{"Count": 0}))
		if err != nil {
			t.Fatal(err)
		}
		if got.Count == nil || *got.Count != 0 {
			t.Errorf("Count = %v, want a pointer to 0", got.Count)
		}
		if got.Server != nil || got.Deep != nil {
			t.Errorf("missing keys should stay nil: %+v", got)
		}
	})

	t.Run("null keeps the pointer nil", func(t *testing.T) {
		got, err := Decode[pointerConfig](configWith(map[string]any{"Count": nil, "Server": nil}))
		if err != nil {
			t.Fatal(err)
		}
		if got.Count != nil || got.Server != nil {
			t.Errorf("got %+v, want nil pointers", got)
		}
	})

	t.Run("type mismatch through a pointer", func(t *testing.T) {
		_, err := Decode[pointerConfig](configWith(map[string]any{"Count": "x"}))
		assertBindError(t, err, ErrTypeMismatch, "Count")
	})
}

type collectionConfig struct {
	Names    []string
	Servers  []serverSection
	Mixed    []any
	Pair     [2]int
	Ports    map[string]int
	Sections map[string]serverSection
}

func TestBindCollections(t *testing.T) {
	runBindCases(t, []bindCase[collectionConfig]{
		{
			name: "every collection kind",
			data: map[string]any{
				"Names": []any{"a", "b"},
				"Servers": []any{
					map[string]any{"Name": "x", "Port": 1},
					map[string]any{"Name": "y"},
				},
				"Mixed":    []any{1, "x", true, map[string]any{"k": []any{2}}},
				"Pair":     []any{1, 2},
				"Ports":    map[string]any{"a": 1, "b": 2},
				"Sections": map[string]any{"s": map[string]any{"Port": 3}},
			},
			want: collectionConfig{
				Names:    []string{"a", "b"},
				Servers:  []serverSection{{Name: "x", Port: 1}, {Name: "y"}},
				Mixed:    []any{1, "x", true, map[string]any{"k": []any{2}}},
				Pair:     [2]int{1, 2},
				Ports:    map[string]int{"a": 1, "b": 2},
				Sections: map[string]serverSection{"s": {Port: 3}},
			},
		},
		{
			name: "typed string slice source",
			data: map[string]any{"Names": []string{"a", "b"}},
			want: collectionConfig{Names: []string{"a", "b"}},
		},
		{
			name: "empty collections are not nil",
			data: map[string]any{"Names": []any{}, "Ports": map[string]any{}},
			want: collectionConfig{Names: []string{}, Ports: map[string]int{}},
		},
		{
			name:     "array with too few items",
			data:     map[string]any{"Pair": []any{1}},
			wantErr:  ErrTypeMismatch,
			wantPath: "Pair",
		},
		{
			name:     "array with too many items",
			data:     map[string]any{"Pair": []any{1, 2, 3}},
			wantErr:  ErrTypeMismatch,
			wantPath: "Pair",
		},
		{
			name:     "bad element in a scalar slice",
			data:     map[string]any{"Names": []any{"a", 2}},
			wantErr:  ErrTypeMismatch,
			wantPath: "Names[1]",
		},
		{
			name:     "bad map value",
			data:     map[string]any{"Ports": map[string]any{"a": "x"}},
			wantErr:  ErrTypeMismatch,
			wantPath: "Ports.a",
		},
		{
			name:     "slice given a map",
			data:     map[string]any{"Names": map[string]any{"a": "b"}},
			wantErr:  ErrTypeMismatch,
			wantPath: "Names",
		},
	})
}

type namedMode string

type namedLevel int

type namedTypesConfig struct {
	Mode  namedMode
	Level namedLevel
	Keys  map[namedMode]int
}

func TestBindNamedTypes(t *testing.T) {
	runBindCases(t, []bindCase[namedTypesConfig]{
		{
			name: "named string and integer types",
			data: map[string]any{"Mode": "fast", "Level": 3, "Keys": map[string]any{"fast": 1}},
			want: namedTypesConfig{Mode: "fast", Level: 3, Keys: map[namedMode]int{"fast": 1}},
		},
	})
}

func TestBindFromJSONNumbers(t *testing.T) {
	cfg := New()
	if err := cfg.FromJSON(`{"Server":{"Port":8080,"Timeout":15.0,"Name":"app"},"Database":{"Port":5432}}`); err != nil {
		t.Fatal(err)
	}

	got, err := Decode[fixtureConfig](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	want := fixtureConfig{
		Server:   serverSection{Name: "app", Port: 8080, Timeout: 15},
		Database: databaseSection{Port: 5432},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}

func TestBindFromJSONRejectsFractionIntoInteger(t *testing.T) {
	cfg := New()
	if err := cfg.FromJSON(`{"Server":{"Port":8080.5}}`); err != nil {
		t.Fatal(err)
	}

	_, err := Decode[fixtureConfig](cfg)

	assertBindError(t, err, ErrTypeMismatch, "Server.Port")
}

func TestBindUnsupportedKinds(t *testing.T) {
	t.Run("channel", func(t *testing.T) {
		_, err := Decode[struct{ C chan int }](configWith(map[string]any{"C": 1}))
		assertBindError(t, err, ErrUnsupportedType, "C")
	})

	t.Run("function", func(t *testing.T) {
		_, err := Decode[struct{ F func() }](configWith(map[string]any{"F": 1}))
		assertBindError(t, err, ErrUnsupportedType, "F")
	})

	t.Run("complex", func(t *testing.T) {
		_, err := Decode[struct{ Z complex128 }](configWith(map[string]any{"Z": 1}))
		assertBindError(t, err, ErrUnsupportedType, "Z")
	})

	t.Run("non-empty interface", func(t *testing.T) {
		_, err := Decode[struct{ E error }](configWith(map[string]any{"E": "boom"}))
		assertBindError(t, err, ErrUnsupportedType, "E")
	})

	t.Run("map with integer keys", func(t *testing.T) {
		_, err := Decode[struct{ M map[int]string }](configWith(map[string]any{"M": map[string]any{"1": "a"}}))
		assertBindError(t, err, ErrUnsupportedType, "M")
	})

	t.Run("absent keys are not reported", func(t *testing.T) {
		if _, err := Decode[struct{ C chan int }](configWith(map[string]any{})); err != nil {
			t.Errorf("err = %v, want nil when the key is absent", err)
		}
		if _, err := Decode[struct{ C chan int }](configWith(map[string]any{"C": nil})); err != nil {
			t.Errorf("err = %v, want nil when the key is null", err)
		}
	})

	t.Run("unsupported source value in an any target", func(t *testing.T) {
		_, err := Decode[map[string]any](configWith(map[string]any{"x": struct{}{}}))
		assertBindError(t, err, ErrUnsupportedType, "x")
	})
}

type chainNode struct {
	Value int
	Next  *chainNode
}

func TestBindDepthLimits(t *testing.T) {
	t.Run("deeply nested data into any", func(t *testing.T) {
		root := map[string]any{}
		current := root
		for i := 0; i < maxBindDepth*2; i++ {
			next := map[string]any{}
			current["n"] = next
			current = next
		}

		_, err := Decode[map[string]any](configWith(root))

		if !errors.Is(err, ErrTooDeep) {
			t.Errorf("err = %v, want ErrTooDeep", err)
		}
	})

	t.Run("self referencing map into any", func(t *testing.T) {
		cyclic := map[string]any{}
		cyclic["self"] = cyclic

		_, err := Decode[map[string]any](configWith(cyclic))

		if !errors.Is(err, ErrTooDeep) {
			t.Errorf("err = %v, want ErrTooDeep", err)
		}
	})

	t.Run("self referencing map into a recursive type", func(t *testing.T) {
		cyclic := map[string]any{"Value": 1}
		cyclic["Next"] = cyclic

		_, err := Decode[chainNode](configWith(cyclic))

		if !errors.Is(err, ErrTooDeep) {
			t.Errorf("err = %v, want ErrTooDeep", err)
		}
	})

	t.Run("self referencing list", func(t *testing.T) {
		cyclic := make([]any, 1)
		cyclic[0] = cyclic

		_, err := Decode[map[string]any](configWith(map[string]any{"list": cyclic}))

		if !errors.Is(err, ErrTooDeep) {
			t.Errorf("err = %v, want ErrTooDeep", err)
		}
	})

	t.Run("recursive type with a short chain", func(t *testing.T) {
		got, err := Decode[chainNode](configWith(map[string]any{
			"Value": 1,
			"Next": map[string]any{
				"Value": 2,
				"Next":  map[string]any{"Value": 3},
			},
		}))
		if err != nil {
			t.Fatal(err)
		}

		want := chainNode{Value: 1, Next: &chainNode{Value: 2, Next: &chainNode{Value: 3}}}
		if !reflect.DeepEqual(*got, want) {
			t.Errorf("got %+v, want %+v", *got, want)
		}
	})
}

type fuzzTarget struct {
	Text     string
	Flag     bool
	Small    int8
	Big      int64
	Unsigned uint32
	Single   float32
	Double   float64
	Pointer  *int
	Nested   *fuzzTarget
	Items    []fuzzTarget
	Pair     [2]string
	Lookup   map[string]float64
	Any      any
	Mixed    []any
}

func FuzzDecodeFromJSON(f *testing.F) {
	f.Add(`{"Text":"a","Flag":true,"Small":1,"Big":2,"Unsigned":3,"Single":1.5,"Double":2.5}`)
	f.Add(`{"Small":300,"Unsigned":-1,"Pair":["a"],"Lookup":{"x":"y"}}`)
	f.Add(`{"Nested":{"Nested":{"Items":[{"Pointer":1}]}},"Any":[1,{"k":null}],"Mixed":[[[]]]}`)
	f.Add(`{"Big":1e30,"Double":"x","Items":{"a":1}}`)
	f.Add(`null`)
	f.Add(`{}`)

	f.Fuzz(func(t *testing.T, input string) {
		cfg := New()
		if err := cfg.FromJSON(input); err != nil {
			return
		}
		Decode[fuzzTarget](cfg)
		Decode[map[string]any](cfg)
		Decode[any](cfg)
		Decode[holder[int]](cfg)
	})
}

func BenchmarkUnmarshal(b *testing.B) {
	content, err := os.ReadFile("../testdata/config.dml")
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Unmarshal[fixtureConfig](content); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	content, err := os.ReadFile("../testdata/config.dml")
	if err != nil {
		b.Fatal(err)
	}
	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Decode[fixtureConfig](cfg); err != nil {
			b.Fatal(err)
		}
	}
}
