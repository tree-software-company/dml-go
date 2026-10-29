package dml

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	maxBindDepth         = 128
	maxPathSegmentLength = 64
	maxExactFloat32Int   = 1 << 24
	maxExactFloat64Int   = 1 << 53
	twoPow63             = float64(1 << 63)
	twoPow64             = float64(1 << 64)
	structTagName        = "dml"
	requiredTagOption    = "required"
	skipTagValue         = "-"
)

type bindField struct {
	index    int
	name     string
	required bool
}

func bindValue(dst, src reflect.Value, path string, depth int) error {
	if depth > maxBindDepth {
		return newBindError(path, ErrTooDeep, "")
	}

	src = unwrapSource(src)
	if !src.IsValid() {
		return nil
	}

	switch dst.Kind() {
	case reflect.Bool:
		return bindBool(dst, src, path)
	case reflect.String:
		return bindString(dst, src, path)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return bindInteger(dst, src, path)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return bindUnsigned(dst, src, path)
	case reflect.Float32, reflect.Float64:
		return bindFloat(dst, src, path)
	case reflect.Struct:
		return bindStruct(dst, src, path, depth)
	case reflect.Pointer:
		return bindPointer(dst, src, path, depth)
	case reflect.Slice:
		return bindSlice(dst, src, path, depth)
	case reflect.Array:
		return bindArray(dst, src, path, depth)
	case reflect.Map:
		return bindMap(dst, src, path, depth)
	case reflect.Interface:
		return bindInterface(dst, src, path, depth)
	default:
		return newBindError(path, ErrUnsupportedType, dst.Type().String())
	}
}

func unwrapSource(src reflect.Value) reflect.Value {
	for i := 0; i < maxBindDepth && src.IsValid(); i++ {
		if src.Kind() != reflect.Interface && src.Kind() != reflect.Pointer {
			return src
		}
		if src.IsNil() {
			return reflect.Value{}
		}
		src = src.Elem()
	}
	return src
}

func bindBool(dst, src reflect.Value, path string) error {
	if src.Kind() != reflect.Bool {
		return mismatch(dst, src, path)
	}
	dst.SetBool(src.Bool())
	return nil
}

func bindString(dst, src reflect.Value, path string) error {
	if src.Kind() != reflect.String {
		return mismatch(dst, src, path)
	}
	dst.SetString(src.String())
	return nil
}

func bindInteger(dst, src reflect.Value, path string) error {
	var value int64

	switch {
	case isSignedKind(src.Kind()):
		value = src.Int()
	case isUnsignedKind(src.Kind()):
		unsigned := src.Uint()
		if unsigned > math.MaxInt64 {
			return outOfRange(dst, path)
		}
		value = int64(unsigned)
	case isFloatKind(src.Kind()):
		number := src.Float()
		if !isIntegral(number) {
			return newBindError(path, ErrTypeMismatch, fmt.Sprintf("expected %s, got a non-integral number", dst.Type()))
		}
		if number < -twoPow63 || number >= twoPow63 {
			return outOfRange(dst, path)
		}
		value = int64(number)
	default:
		return mismatch(dst, src, path)
	}

	if dst.OverflowInt(value) {
		return outOfRange(dst, path)
	}
	dst.SetInt(value)
	return nil
}

func bindUnsigned(dst, src reflect.Value, path string) error {
	var value uint64

	switch {
	case isSignedKind(src.Kind()):
		signed := src.Int()
		if signed < 0 {
			return outOfRange(dst, path)
		}
		value = uint64(signed)
	case isUnsignedKind(src.Kind()):
		value = src.Uint()
	case isFloatKind(src.Kind()):
		number := src.Float()
		if !isIntegral(number) {
			return newBindError(path, ErrTypeMismatch, fmt.Sprintf("expected %s, got a non-integral number", dst.Type()))
		}
		if number < 0 || number >= twoPow64 {
			return outOfRange(dst, path)
		}
		value = uint64(number)
	default:
		return mismatch(dst, src, path)
	}

	if dst.OverflowUint(value) {
		return outOfRange(dst, path)
	}
	dst.SetUint(value)
	return nil
}

func bindFloat(dst, src reflect.Value, path string) error {
	limit := int64(maxExactFloat64Int)
	if dst.Type().Bits() == 32 {
		limit = maxExactFloat32Int
	}

	var value float64

	switch {
	case isSignedKind(src.Kind()):
		signed := src.Int()
		if signed > limit || signed < -limit {
			return outOfRange(dst, path)
		}
		value = float64(signed)
	case isUnsignedKind(src.Kind()):
		unsigned := src.Uint()
		if unsigned > uint64(limit) {
			return outOfRange(dst, path)
		}
		value = float64(unsigned)
	case isFloatKind(src.Kind()):
		value = src.Float()
		if dst.OverflowFloat(value) {
			return outOfRange(dst, path)
		}
	default:
		return mismatch(dst, src, path)
	}

	dst.SetFloat(value)
	return nil
}

func bindStruct(dst, src reflect.Value, path string, depth int) error {
	if !isStringKeyedMap(src) {
		return mismatch(dst, src, path)
	}

	entries, names := collectEntries(src)
	for _, field := range structFields(dst.Type()) {
		fieldPath := joinFieldPath(path, field.name)
		value, found := lookupEntry(entries, names, field.name)
		if !found || !unwrapSource(value).IsValid() {
			if field.required {
				return newBindError(fieldPath, ErrMissingRequired, "")
			}
			continue
		}
		if err := bindValue(dst.Field(field.index), value, fieldPath, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func bindPointer(dst, src reflect.Value, path string, depth int) error {
	target := reflect.New(dst.Type().Elem())
	if err := bindValue(target.Elem(), src, path, depth+1); err != nil {
		return err
	}
	dst.Set(target)
	return nil
}

func bindSlice(dst, src reflect.Value, path string, depth int) error {
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return mismatch(dst, src, path)
	}

	length := src.Len()
	items := reflect.MakeSlice(dst.Type(), length, length)
	for i := 0; i < length; i++ {
		if err := bindValue(items.Index(i), src.Index(i), indexPath(path, i), depth+1); err != nil {
			return err
		}
	}
	dst.Set(items)
	return nil
}

func bindArray(dst, src reflect.Value, path string, depth int) error {
	if src.Kind() != reflect.Slice && src.Kind() != reflect.Array {
		return mismatch(dst, src, path)
	}
	if src.Len() != dst.Len() {
		detail := fmt.Sprintf("expected %d items, got %d", dst.Len(), src.Len())
		return newBindError(path, ErrTypeMismatch, detail)
	}

	for i := 0; i < src.Len(); i++ {
		if err := bindValue(dst.Index(i), src.Index(i), indexPath(path, i), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func bindMap(dst, src reflect.Value, path string, depth int) error {
	keyType := dst.Type().Key()
	if keyType.Kind() != reflect.String {
		return newBindError(path, ErrUnsupportedType, dst.Type().String())
	}
	if !isStringKeyedMap(src) {
		return mismatch(dst, src, path)
	}

	entries, names := collectEntries(src)
	items := reflect.MakeMapWithSize(dst.Type(), len(names))
	for _, name := range names {
		item := reflect.New(dst.Type().Elem()).Elem()
		if err := bindValue(item, entries[name], joinDataPath(path, name), depth+1); err != nil {
			return err
		}
		items.SetMapIndex(reflect.ValueOf(name).Convert(keyType), item)
	}
	dst.Set(items)
	return nil
}

func bindInterface(dst, src reflect.Value, path string, depth int) error {
	if dst.NumMethod() != 0 {
		return newBindError(path, ErrUnsupportedType, dst.Type().String())
	}

	copied, err := copyAny(src, path, depth)
	if err != nil {
		return err
	}
	if copied != nil {
		dst.Set(reflect.ValueOf(copied))
	}
	return nil
}

func copyAny(src reflect.Value, path string, depth int) (any, error) {
	if depth > maxBindDepth {
		return nil, newBindError(path, ErrTooDeep, "")
	}

	src = unwrapSource(src)
	if !src.IsValid() {
		return nil, nil
	}

	switch kind := src.Kind(); {
	case kind == reflect.Bool, kind == reflect.String, isSignedKind(kind), isUnsignedKind(kind), isFloatKind(kind):
		return src.Interface(), nil
	case kind == reflect.Map:
		return copyMapAny(src, path, depth)
	case kind == reflect.Slice, kind == reflect.Array:
		return copyListAny(src, path, depth)
	default:
		return nil, newBindError(path, ErrUnsupportedType, sourceKind(src))
	}
}

func copyMapAny(src reflect.Value, path string, depth int) (any, error) {
	if !isStringKeyedMap(src) {
		return nil, newBindError(path, ErrUnsupportedType, "map with non-string keys")
	}

	entries, names := collectEntries(src)
	items := make(map[string]any, len(names))
	for _, name := range names {
		item, err := copyAny(entries[name], joinDataPath(path, name), depth+1)
		if err != nil {
			return nil, err
		}
		items[name] = item
	}
	return items, nil
}

func copyListAny(src reflect.Value, path string, depth int) (any, error) {
	items := make([]any, src.Len())
	for i := range items {
		item, err := copyAny(src.Index(i), indexPath(path, i), depth+1)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	return items, nil
}

func structFields(t reflect.Type) []bindField {
	fields := make([]bindField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		tag := field.Tag.Get(structTagName)
		if tag == skipTagValue {
			continue
		}

		name, options, _ := strings.Cut(tag, ",")
		if name == "" {
			name = field.Name
		}
		fields = append(fields, bindField{
			index:    i,
			name:     name,
			required: hasTagOption(options, requiredTagOption),
		})
	}
	return fields
}

func hasTagOption(options, wanted string) bool {
	for _, option := range strings.Split(options, ",") {
		if option == wanted {
			return true
		}
	}
	return false
}

func collectEntries(src reflect.Value) (map[string]reflect.Value, []string) {
	entries := make(map[string]reflect.Value, src.Len())
	names := make([]string, 0, src.Len())

	iter := src.MapRange()
	for iter.Next() {
		name := iter.Key().String()
		entries[name] = iter.Value()
		names = append(names, name)
	}
	sort.Strings(names)

	return entries, names
}

func lookupEntry(entries map[string]reflect.Value, names []string, name string) (reflect.Value, bool) {
	if value, found := entries[name]; found {
		return value, true
	}
	for _, candidate := range names {
		if strings.EqualFold(candidate, name) {
			return entries[candidate], true
		}
	}
	return reflect.Value{}, false
}

func isStringKeyedMap(src reflect.Value) bool {
	return src.Kind() == reflect.Map && src.Type().Key().Kind() == reflect.String
}

func isIntegral(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, 0) && number == math.Trunc(number)
}

func isSignedKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	}
	return false
}

func isUnsignedKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

func isFloatKind(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}

func sourceKind(src reflect.Value) string {
	kind := src.Kind()
	switch {
	case kind == reflect.Bool:
		return "bool"
	case kind == reflect.String:
		return "string"
	case isSignedKind(kind), isUnsignedKind(kind), isFloatKind(kind):
		return "number"
	case kind == reflect.Slice, kind == reflect.Array:
		return "list"
	case kind == reflect.Map:
		return "map"
	default:
		return "unsupported"
	}
}

func mismatch(dst, src reflect.Value, path string) error {
	detail := fmt.Sprintf("expected %s, got %s", dst.Type(), sourceKind(src))
	return newBindError(path, ErrTypeMismatch, detail)
}

func outOfRange(dst reflect.Value, path string) error {
	return newBindError(path, ErrOutOfRange, dst.Type().String())
}

func joinFieldPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func joinDataPath(path, key string) string {
	if len(key) > maxPathSegmentLength {
		key = key[:maxPathSegmentLength]
	}
	quoted := strconv.Quote(key)
	return joinFieldPath(path, quoted[1:len(quoted)-1])
}

func indexPath(path string, index int) string {
	return path + "[" + strconv.Itoa(index) + "]"
}
