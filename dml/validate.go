package dml

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	validateTagName = "validate"
	maxViolations   = 100
	ruleRequired    = "required"
	ruleMin         = "min"
	ruleMax         = "max"
	ruleLen         = "len"
	ruleOneOf       = "oneof"
	ruleDive        = "dive"
)

type rule struct {
	name  string
	param string
	elem  []rule
}

type compiledRule struct {
	name  string
	param string
	check func(reflect.Value) bool
}

type fieldRules struct {
	own  []compiledRule
	elem *fieldRules
}

type fieldPlan struct {
	name  string
	index int
	rules fieldRules
}

type validator struct {
	violations []*FieldError
	plans      map[reflect.Type][]fieldPlan
}

func validateDecoded(dst, src reflect.Value) error {
	v := &validator{plans: make(map[reflect.Type][]fieldPlan)}
	if err := v.descend(fieldRules{}, dst, unwrapSource(src), "", 0); err != nil {
		return err
	}
	if len(v.violations) > 0 {
		return &ValidationError{Fields: v.violations}
	}
	return nil
}

func (v *validator) full() bool {
	return len(v.violations) >= maxViolations
}

func (v *validator) add(path, name, param string) {
	if !v.full() {
		v.violations = append(v.violations, newFieldError(path, name, param))
	}
}

func (v *validator) walkValue(rules fieldRules, dst, src reflect.Value, path string, depth int) error {
	src = unwrapSource(src)
	if !src.IsValid() {
		return v.walkAbsent(rules, dst, path, depth)
	}

	if value := indirectValue(dst); value.IsValid() {
		for _, r := range rules.own {
			if !r.check(value) {
				v.add(path, r.name, r.param)
			}
		}
	}
	return v.descend(rules, dst, src, path, depth)
}

func (v *validator) walkAbsent(rules fieldRules, dst reflect.Value, path string, depth int) error {
	for _, r := range rules.own {
		if r.name == ruleRequired {
			v.add(path, ruleRequired, "")
			return nil
		}
	}
	if dst.Kind() == reflect.Struct {
		return v.descend(fieldRules{}, dst, reflect.Value{}, path, depth)
	}
	return nil
}

func (v *validator) descend(rules fieldRules, dst, src reflect.Value, path string, depth int) error {
	if depth > maxBindDepth {
		return newBindError(path, ErrTooDeep, "")
	}

	switch dst.Kind() {
	case reflect.Pointer:
		if dst.IsNil() {
			return nil
		}
		return v.descend(rules, dst.Elem(), src, path, depth+1)
	case reflect.Struct:
		return v.walkStruct(dst, src, path, depth)
	case reflect.Slice, reflect.Array:
		return v.walkList(rules, dst, src, path, depth)
	case reflect.Map:
		return v.walkMap(rules, dst, src, path, depth)
	}
	return nil
}

func (v *validator) walkStruct(dst, src reflect.Value, path string, depth int) error {
	plans, err := v.plan(dst.Type(), path)
	if err != nil {
		return err
	}

	var entries map[string]reflect.Value
	var names []string
	if isStringKeyedMap(src) {
		entries, names = collectEntries(src)
	}

	for _, p := range plans {
		value, _ := lookupEntry(entries, names, p.name)
		if err := v.walkValue(p.rules, dst.Field(p.index), value, joinFieldPath(path, p.name), depth+1); err != nil {
			return err
		}
		if v.full() {
			return nil
		}
	}
	return nil
}

func (v *validator) walkList(rules fieldRules, dst, src reflect.Value, path string, depth int) error {
	elemRules := rules.elemRules()
	if !needsDescent(dst.Type().Elem(), elemRules) {
		return nil
	}

	isList := src.Kind() == reflect.Slice || src.Kind() == reflect.Array
	for i := 0; i < dst.Len(); i++ {
		var item reflect.Value
		if isList && i < src.Len() {
			item = src.Index(i)
		}
		if err := v.walkValue(elemRules, dst.Index(i), item, indexPath(path, i), depth+1); err != nil {
			return err
		}
		if v.full() {
			return nil
		}
	}
	return nil
}

func (v *validator) walkMap(rules fieldRules, dst, src reflect.Value, path string, depth int) error {
	keyType := dst.Type().Key()
	elemRules := rules.elemRules()
	if keyType.Kind() != reflect.String || !isStringKeyedMap(src) || !needsDescent(dst.Type().Elem(), elemRules) {
		return nil
	}

	entries, names := collectEntries(src)
	for _, name := range names {
		item := dst.MapIndex(reflect.ValueOf(name).Convert(keyType))
		if !item.IsValid() {
			continue
		}
		if err := v.walkValue(elemRules, item, entries[name], joinDataPath(path, name), depth+1); err != nil {
			return err
		}
		if v.full() {
			return nil
		}
	}
	return nil
}

func (v *validator) plan(t reflect.Type, path string) ([]fieldPlan, error) {
	if plans, cached := v.plans[t]; cached {
		return plans, nil
	}

	fields := structFields(t)
	plans := make([]fieldPlan, 0, len(fields))
	for _, field := range fields {
		structField := t.Field(field.index)
		rules, err := compileTag(structField.Tag.Get(validateTagName), structField.Type)
		if err != nil {
			return nil, newBindError(joinFieldPath(path, field.name), ErrInvalidRule, err.Error())
		}
		plans = append(plans, fieldPlan{name: field.name, index: field.index, rules: rules})
	}

	v.plans[t] = plans
	return plans, nil
}

func (r fieldRules) elemRules() fieldRules {
	if r.elem == nil {
		return fieldRules{}
	}
	return *r.elem
}

func needsDescent(elem reflect.Type, rules fieldRules) bool {
	if len(rules.own) > 0 || rules.elem != nil {
		return true
	}
	switch indirectType(elem).Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	}
	return false
}

func indirectType(t reflect.Type) reflect.Type {
	for i := 0; i < maxBindDepth && t.Kind() == reflect.Pointer; i++ {
		t = t.Elem()
	}
	return t
}

func indirectValue(value reflect.Value) reflect.Value {
	for i := 0; i < maxBindDepth && value.Kind() == reflect.Pointer; i++ {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func compileTag(tag string, t reflect.Type) (fieldRules, error) {
	if tag == "" {
		return fieldRules{}, nil
	}
	rules, err := parseRules(strings.Split(tag, ","))
	if err != nil {
		return fieldRules{}, err
	}
	return compileRules(rules, t)
}

func parseRules(items []string) ([]rule, error) {
	rules := make([]rule, 0, len(items))
	for i, item := range items {
		name, param, hasParam := strings.Cut(strings.TrimSpace(item), "=")
		switch {
		case name == "":
			return nil, errors.New("empty rule")
		case name == ruleDive:
			if hasParam {
				return nil, errors.New(`rule "dive" takes no parameter`)
			}
			elem, err := parseRules(items[i+1:])
			if err != nil {
				return nil, err
			}
			return append(rules, rule{name: name, elem: elem}), nil
		}
		rules = append(rules, rule{name: name, param: param})
	}
	return rules, nil
}

func compileRules(rules []rule, t reflect.Type) (fieldRules, error) {
	t = indirectType(t)

	var compiled fieldRules
	for _, r := range rules {
		if r.name == ruleDive {
			elemType, err := containerElem(r, t)
			if err != nil {
				return fieldRules{}, err
			}
			elem, err := compileRules(r.elem, elemType)
			if err != nil {
				return fieldRules{}, err
			}
			compiled.elem = &elem
			continue
		}

		c, err := compileRule(r, t)
		if err != nil {
			return fieldRules{}, err
		}
		compiled.own = append(compiled.own, c)
	}
	return compiled, nil
}

func compileRule(r rule, t reflect.Type) (compiledRule, error) {
	compiled := compiledRule{name: r.name, param: r.param}

	var err error
	switch r.name {
	case ruleRequired:
		if r.param != "" {
			return compiledRule{}, fmt.Errorf("rule %q takes no parameter", r.name)
		}
		compiled.check = checkRequired
	case ruleMin, ruleMax:
		compiled.check, err = compileBound(r, t)
	case ruleLen:
		compiled.check, err = compileLength(r, t)
	case ruleOneOf:
		compiled.check, err = compileOneOf(r, t)
	default:
		err = fmt.Errorf("unknown rule %q", r.name)
	}
	if err != nil {
		return compiledRule{}, err
	}
	return compiled, nil
}

func containerElem(r rule, t reflect.Type) (reflect.Type, error) {
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return t.Elem(), nil
	}
	return nil, inapplicable(r.name, t)
}

func checkRequired(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		return value.Len() > 0
	}
	return true
}

func compileBound(r rule, t reflect.Type) (func(reflect.Value) bool, error) {
	upper := r.name == ruleMax
	kind := t.Kind()

	switch {
	case kind == reflect.String:
		limit, err := parseLength(r)
		if err != nil {
			return nil, err
		}
		return func(value reflect.Value) bool {
			return withinBound(utf8.RuneCountInString(value.String()), limit, upper)
		}, nil
	case kind == reflect.Slice, kind == reflect.Array, kind == reflect.Map:
		limit, err := parseLength(r)
		if err != nil {
			return nil, err
		}
		return func(value reflect.Value) bool {
			return withinBound(value.Len(), limit, upper)
		}, nil
	case isSignedKind(kind):
		limit, err := strconv.ParseInt(r.param, 10, 64)
		if err != nil {
			return nil, needsNumber(r)
		}
		return func(value reflect.Value) bool {
			return withinBound(value.Int(), limit, upper)
		}, nil
	case isUnsignedKind(kind):
		limit, err := strconv.ParseUint(r.param, 10, 64)
		if err != nil {
			return nil, needsNumber(r)
		}
		return func(value reflect.Value) bool {
			return withinBound(value.Uint(), limit, upper)
		}, nil
	case isFloatKind(kind):
		limit, err := strconv.ParseFloat(r.param, t.Bits())
		if err != nil || math.IsNaN(limit) || math.IsInf(limit, 0) {
			return nil, needsNumber(r)
		}
		return func(value reflect.Value) bool {
			return withinBound(value.Float(), limit, upper)
		}, nil
	}
	return nil, inapplicable(r.name, t)
}

func compileLength(r rule, t reflect.Type) (func(reflect.Value) bool, error) {
	kind := t.Kind()
	if kind != reflect.String && kind != reflect.Slice && kind != reflect.Array && kind != reflect.Map {
		return nil, inapplicable(r.name, t)
	}

	want, err := parseLength(r)
	if err != nil {
		return nil, err
	}
	if kind == reflect.String {
		return func(value reflect.Value) bool {
			return utf8.RuneCountInString(value.String()) == want
		}, nil
	}
	return func(value reflect.Value) bool {
		return value.Len() == want
	}, nil
}

func compileOneOf(r rule, t reflect.Type) (func(reflect.Value) bool, error) {
	options := strings.Fields(r.param)
	if len(options) == 0 {
		return nil, fmt.Errorf("rule %q needs at least one value", r.name)
	}

	kind := t.Kind()
	switch {
	case kind == reflect.String:
		allowed := make(map[string]struct{}, len(options))
		for _, option := range options {
			allowed[option] = struct{}{}
		}
		return func(value reflect.Value) bool {
			_, ok := allowed[value.String()]
			return ok
		}, nil
	case isSignedKind(kind):
		allowed, err := numericOptions(r, options, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		})
		if err != nil {
			return nil, err
		}
		return func(value reflect.Value) bool {
			_, ok := allowed[value.Int()]
			return ok
		}, nil
	case isUnsignedKind(kind):
		allowed, err := numericOptions(r, options, func(s string) (uint64, error) {
			return strconv.ParseUint(s, 10, 64)
		})
		if err != nil {
			return nil, err
		}
		return func(value reflect.Value) bool {
			_, ok := allowed[value.Uint()]
			return ok
		}, nil
	}
	return nil, inapplicable(r.name, t)
}

func numericOptions[N int64 | uint64](r rule, options []string, parse func(string) (N, error)) (map[N]struct{}, error) {
	allowed := make(map[N]struct{}, len(options))
	for _, option := range options {
		number, err := parse(option)
		if err != nil {
			return nil, fmt.Errorf("rule %q needs whole numbers for this field", r.name)
		}
		allowed[number] = struct{}{}
	}
	return allowed, nil
}

func withinBound[N cmp.Ordered](value, limit N, upper bool) bool {
	if upper {
		return value <= limit
	}
	return value >= limit
}

func parseLength(r rule) (int, error) {
	length, err := strconv.Atoi(r.param)
	if err != nil || length < 0 {
		return 0, fmt.Errorf("rule %q needs a non-negative whole number", r.name)
	}
	return length, nil
}

func needsNumber(r rule) error {
	return fmt.Errorf("rule %q needs a number that fits the field type", r.name)
}

func inapplicable(name string, t reflect.Type) error {
	return fmt.Errorf("rule %q cannot be used on %s", name, t)
}
