package netlink

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/mdlayher/netlink/nlenc"
)

type SubMessageResolver func(selector any) (any, error)

// AttributeCodecConfig holds optional configuration for a Codec.
type AttributeCodecConfig struct {
	// Resolvers map sub-message tag names to functions that return
	// a concrete struct pointer for a given selector value.
	Resolvers map[string]SubMessageResolver
}

// Codec is a stateful encoder/decoder that holds configuration.
type Codec struct {
	resolvers map[string]SubMessageResolver
}

// NewAttributeCodec creates a new Codec with the given configuration.
// Pass nil for families that need no special configuration.
func NewAttributeCodec(cfg *AttributeCodecConfig) *Codec {
	c := &Codec{}
	if cfg != nil {
		c.resolvers = cfg.Resolvers
	}
	return c
}

// Marshal encodes a struct into netlink attribute bytes.
func (c *Codec) Marshal(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("netlink: Marshal requires a struct or pointer to struct, got %T", v)
	}
	attrs, err := c.encodeAttrs(rv)
	if err != nil {
		return nil, err
	}
	return MarshalAttributes(attrs)
}

// Unmarshal decodes netlink attribute bytes into a struct.
func (c *Codec) Unmarshal(b []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("netlink: Unmarshal requires a non-nil pointer to struct, got %T", v)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("netlink: Unmarshal requires a pointer to struct, got %T", v)
	}
	attrs, err := UnmarshalAttributes(b)
	if err != nil {
		return err
	}
	return c.decodeAttrs(attrs, rv)
}

type attrType uint8

const (
	_ attrType = iota
	attrScalar
	attrString
	attrBinary
	attrFlag
	attrNest
)

type attrLayout uint8

const (
	layoutSingle  attrLayout = iota // single value
	layoutMulti                     // repeated attribute (same index)
	layoutIndexed                   // nested with sequential uint16 indices
)

type fieldTag struct {
	attrIndex uint16
	byteOrder binary.ByteOrder
	submsg    string
	selector  string
}

type fieldInfo struct {
	index  int
	typ    attrType
	layout attrLayout
	kind   reflect.Kind
	tag    fieldTag
}

type structFields struct {
	fields []fieldInfo
	byAttr map[uint16]*fieldInfo
}

// cache of structFields by struct type for efficient lookup
var cache sync.Map

func typeFields(rv reflect.Value) (*structFields, error) {
	t := rv.Type()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if v, ok := cache.Load(t); ok {
		return v.(*structFields), nil
	}
	sf := &structFields{}
	// Parse fields and check for duplicate attribute indices.
	seen := make(map[uint16]int)
	for i := 0; i < rv.Type().NumField(); i++ {
		fi, err := parseField(rv, i)
		if err != nil {
			return nil, err
		}
		if fi == nil {
			continue
		}
		if prev, dup := seen[fi.tag.attrIndex]; dup {
			return nil, fmt.Errorf("netlink: %s: duplicate attribute index %d (also used by field %d)",
				rv.Type().Field(i).Name, fi.tag.attrIndex, prev)
		}
		seen[fi.tag.attrIndex] = fi.index
		sf.fields = append(sf.fields, *fi)
	}
	// Build byAttr after the slice is complete so pointers remain stable.
	sf.byAttr = make(map[uint16]*fieldInfo, len(sf.fields))
	for i := range sf.fields {
		sf.byAttr[sf.fields[i].tag.attrIndex] = &sf.fields[i]
	}
	v, _ := cache.LoadOrStore(t, sf)
	return v.(*structFields), nil
}

func parseField(parent reflect.Value, idx int) (*fieldInfo, error) {
	sf := parent.Type().Field(idx)
	name := parent.Type().Name() + "." + sf.Name

	if !sf.IsExported() {
		return nil, nil
	}

	tag := sf.Tag.Get("netlink")
	if tag == "" || tag == "-" {
		return nil, nil
	}

	fi := fieldInfo{
		index: idx,
		tag: fieldTag{
			byteOrder: binary.NativeEndian,
		},
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("netlink: %s: empty tag", name)
	}

	n, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("netlink: %s: invalid attribute index %q", name, parts[0])
	}
	fi.tag.attrIndex = uint16(n)

	if err := parseTagOptions(&fi, parts[1:], sf, parent); err != nil {
		return nil, err
	}

	// Validate submsg/selector pairing.
	if fi.tag.submsg != "" && fi.tag.selector == "" {
		return nil, fmt.Errorf("netlink: %s: submsg requires a selector tag", name)
	}
	if fi.tag.selector != "" && fi.tag.submsg == "" {
		return nil, fmt.Errorf("netlink: %s: selector requires a submsg tag", name)
	}

	// Determine base type.
	fi.typ, err = inferType(sf.Type)
	if err != nil {
		return nil, err
	}

	// Determine layout.
	// When multi or indexed is explicitly tagged on []uint8, treat elements
	// as individual scalars rather than a binary blob.
	if fi.layout == layoutIndexed {
		if fi.typ == attrBinary {
			fi.typ = attrScalar
		}
	} else if fi.layout == layoutMulti || (sf.Type.Kind() == reflect.Slice && fi.typ != attrBinary) {
		fi.layout = layoutMulti
		if fi.typ == attrBinary {
			fi.typ = attrScalar
		}
	}

	t := sf.Type
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if isScalarKind(t.Kind()) {
		fi.kind = t.Kind()
	}

	return &fi, nil
}

func parseTagOptions(fi *fieldInfo, opts []string, sf reflect.StructField, parent reflect.Value) error {
	name := parent.Type().Name() + "." + sf.Name
	for _, opt := range opts {
		switch {
		case opt == "be":
			fi.tag.byteOrder = binary.BigEndian
			if !isScalarKind(sf.Type.Kind()) {
				return fmt.Errorf("netlink: %s: byte order tag requires a scalar type, got %s", name, sf.Type)
			}
		case opt == "le":
			fi.tag.byteOrder = binary.LittleEndian
			if !isScalarKind(sf.Type.Kind()) {
				return fmt.Errorf("netlink: %s: byte order tag requires a scalar type, got %s", name, sf.Type)
			}
		case opt == "indexed":
			if sf.Type.Kind() != reflect.Slice {
				return fmt.Errorf("netlink: %s: indexed requires a slice type, got %s", name, sf.Type)
			}
			fi.layout = layoutIndexed
		case opt == "multi":
			fi.layout = layoutMulti
		case strings.HasPrefix(opt, "submsg="):
			fi.tag.submsg = strings.TrimPrefix(opt, "submsg=")
			if fi.tag.submsg == "" {
				return fmt.Errorf("netlink: %s: submsg tag requires a value", name)
			}
			if sf.Type.Kind() != reflect.Interface {
				return fmt.Errorf("netlink: %s: submsg requires an interface type, got %s", name, sf.Type)
			}
		case strings.HasPrefix(opt, "selector="):
			fi.tag.selector = strings.TrimPrefix(opt, "selector=")
			if fi.tag.selector == "" {
				return fmt.Errorf("netlink: %s: selector tag requires a value", name)
			}
			selSf, ok := parent.Type().FieldByName(fi.tag.selector)
			if !ok {
				return fmt.Errorf("netlink: %s: selector references unknown field %q", name, fi.tag.selector)
			}
			k := selSf.Type.Kind()
			if !selSf.IsExported() || (k != reflect.String && !isScalarKind(k)) {
				return fmt.Errorf("netlink: %s: selector field %q must be an exported string or integer", name, fi.tag.selector)
			}
		default:
			return fmt.Errorf("netlink: %s: unknown tag option %q", name, opt)
		}
	}
	return nil
}

// inferType determines the base attribute type from a Go reflect.Type.
// It unwraps pointers and slices to find the underlying value type.
func inferType(t reflect.Type) (attrType, error) {
	// Unwrap pointer.
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Unwrap slice — special case []byte as binary.
	if t.Kind() == reflect.Slice {
		if t.Elem().Kind() == reflect.Uint8 {
			return attrBinary, nil
		}
		t = t.Elem()
		// Unwrap *T inside []* T.
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
	}

	switch {
	case t.Kind() == reflect.Struct || t.Kind() == reflect.Interface:
		return attrNest, nil
	case t.Kind() == reflect.String:
		return attrString, nil
	case t.Kind() == reflect.Bool:
		return attrFlag, nil
	case isScalarKind(t.Kind()):
		return attrScalar, nil
	default:
		return 0, fmt.Errorf("netlink: unsupported field type %s", t)
	}
}

func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func (c *Codec) encodeAttrs(rv reflect.Value) ([]Attribute, error) {
	sf, err := typeFields(rv)
	if err != nil {
		return nil, err
	}
	var attrs []Attribute
	for i := range sf.fields {
		fi := &sf.fields[i]
		a, err := c.encodeField(rv.Field(fi.index), fi)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, a...)
	}
	return attrs, nil
}

func (c *Codec) decodeAttrs(attrs []Attribute, rv reflect.Value) error {
	sf, err := typeFields(rv)
	if err != nil {
		return err
	}

	// Decode non-submsg fields first so selector values are populated.
	for _, a := range attrs {
		typ := a.Type & ^uint16(Nested|NetByteOrder)
		fi, ok := sf.byAttr[typ]
		if !ok || fi.tag.submsg != "" {
			continue
		}
		if err := c.decodeField(a.Data, rv, fi); err != nil {
			return err
		}
	}
	// Now decode submsg fields that depend on selectors.
	for _, a := range attrs {
		typ := a.Type & ^uint16(Nested|NetByteOrder)
		fi, ok := sf.byAttr[typ]
		if !ok || fi.tag.submsg == "" {
			continue
		}
		if err := c.decodeField(a.Data, rv, fi); err != nil {
			return err
		}
	}
	return nil
}

func (c *Codec) encodeField(fv reflect.Value, fi *fieldInfo) ([]Attribute, error) {
	if fi.tag.submsg != "" {
		return c.encodeSubMessage(fv, fi)
	}
	if fi.layout == layoutIndexed {
		return c.encodeIndexed(fv, fi)
	}
	if fi.layout == layoutMulti {
		return c.encodeMulti(fv, fi)
	}
	a, err := c.encodeValue(fv, fi)
	if err != nil {
		return nil, err
	}
	if a.Data == nil {
		return nil, nil
	}
	return []Attribute{a}, nil
}

func (c *Codec) decodeField(data []byte, rv reflect.Value, fi *fieldInfo) error {
	fv := rv.Field(fi.index)
	if fi.tag.submsg != "" {
		sel := rv.FieldByName(fi.tag.selector).Interface()
		return c.decodeSubMessage(data, fv, fi, sel)
	}
	if fi.layout == layoutIndexed {
		return c.decodeIndexed(data, fv, fi)
	}
	if fi.layout == layoutMulti {
		return c.decodeMulti(data, fv, fi)
	}
	return c.decodeValue(data, fv, fi)
}

func (c *Codec) encodeValue(fv reflect.Value, fi *fieldInfo) (Attribute, error) {
	typ := fi.tag.attrIndex
	if fi.typ == attrNest {
		typ = Nested | fi.tag.attrIndex
	}
	fv, ok := unwrapValue(fv)
	if !ok {
		return Attribute{}, nil
	}
	var data []byte
	var err error
	switch fi.typ {
	case attrScalar:
		data = putScalar(fv, fi)
	case attrString:
		if fv.String() == "" {
			return Attribute{}, nil
		}
		data = nlenc.Bytes(fv.String())
	case attrBinary:
		if fv.IsNil() {
			return Attribute{}, nil
		}
		data = fv.Bytes()
	case attrFlag:
		if !fv.Bool() {
			return Attribute{}, nil
		}
		data = []byte{}
	case attrNest:
		var children []Attribute
		children, err = c.encodeAttrs(fv)
		if err != nil {
			return Attribute{}, err
		}
		data, err = MarshalAttributes(children)
		if err != nil {
			return Attribute{}, err
		}
	}
	return Attribute{Type: typ, Data: data}, nil
}

func (c *Codec) decodeValue(data []byte, fv reflect.Value, fi *fieldInfo) error {
	switch fi.typ {
	case attrScalar:
		if err := setScalar(data, fv, fi); err != nil {
			return err
		}
	case attrString:
		s := nlenc.String(data)
		if fv.Kind() == reflect.Pointer {
			fv.Set(reflect.ValueOf(&s))
		} else {
			fv.SetString(s)
		}
	case attrBinary:
		fv.SetBytes(append([]byte(nil), data...))
	case attrFlag:
		fv.SetBool(true)
	case attrNest:
		var target reflect.Value
		if fv.Kind() == reflect.Pointer {
			target = reflect.New(fv.Type().Elem())
		} else {
			target = fv.Addr()
		}
		children, err := UnmarshalAttributes(data)
		if err != nil {
			return err
		}
		if err := c.decodeAttrs(children, target.Elem()); err != nil {
			return err
		}
		if fv.Kind() == reflect.Pointer {
			fv.Set(target)
		}
	}
	return nil
}

// unwrapValue dereferences pointers and interfaces to reach the concrete value.
// Returns false if a nil is encountered at any level.
func unwrapValue(fv reflect.Value) (reflect.Value, bool) {
	for fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Interface {
		if fv.IsNil() {
			return fv, false
		}
		fv = fv.Elem()
	}
	return fv, true
}

func (c *Codec) encodeMulti(fv reflect.Value, fi *fieldInfo) ([]Attribute, error) {
	var attrs []Attribute
	for i := 0; i < fv.Len(); i++ {
		a, err := c.encodeValue(fv.Index(i), fi)
		if err != nil {
			return nil, err
		}
		if a.Data == nil {
			continue
		}
		attrs = append(attrs, a)
	}
	return attrs, nil
}

func (c *Codec) decodeMulti(data []byte, fv reflect.Value, fi *fieldInfo) error {
	elemType := fv.Type().Elem()
	isPtr := elemType.Kind() == reflect.Pointer
	baseType := elemType
	if isPtr {
		baseType = elemType.Elem()
	}
	elem := reflect.New(baseType)
	if err := c.decodeValue(data, elem.Elem(), fi); err != nil {
		return err
	}
	if isPtr {
		fv.Set(reflect.Append(fv, elem))
	} else {
		fv.Set(reflect.Append(fv, elem.Elem()))
	}
	return nil
}

func (c *Codec) encodeIndexed(fv reflect.Value, fi *fieldInfo) ([]Attribute, error) {
	if fv.Len() == 0 {
		return nil, nil
	}
	var children []Attribute
	for i := 0; i < fv.Len(); i++ {
		a, err := c.encodeValue(fv.Index(i), fi)
		if err != nil {
			return nil, err
		}
		// Override the attribute index with the sequential position.
		a.Type = uint16(i)
		if fi.typ == attrNest {
			a.Type |= Nested
		}
		children = append(children, a)
	}
	data, err := MarshalAttributes(children)
	if err != nil {
		return nil, err
	}
	return []Attribute{{Type: Nested | fi.tag.attrIndex, Data: data}}, nil
}

func (c *Codec) decodeIndexed(data []byte, fv reflect.Value, fi *fieldInfo) error {
	elemType := fv.Type().Elem()
	children, err := UnmarshalAttributes(data)
	if err != nil {
		return err
	}
	for _, child := range children {
		elem := reflect.New(elemType)
		if err := c.decodeValue(child.Data, elem.Elem(), fi); err != nil {
			return err
		}
		fv.Set(reflect.Append(fv, elem.Elem()))
	}
	return nil
}

func scalarSize(k reflect.Kind) int {
	switch k {
	case reflect.Uint8, reflect.Int8:
		return 1
	case reflect.Uint16, reflect.Int16:
		return 2
	case reflect.Uint32, reflect.Int32:
		return 4
	case reflect.Uint64, reflect.Int64:
		return 8
	default:
		return 0
	}
}

func putScalar(val reflect.Value, fi *fieldInfo) []byte {
	var raw uint64
	if isSignedKind(fi.kind) {
		raw = uint64(val.Int())
	} else {
		raw = val.Uint()
	}
	sz := scalarSize(fi.kind)
	if sz == 1 {
		return []byte{byte(raw)}
	}
	b := make([]byte, sz)
	switch sz {
	case 2:
		fi.tag.byteOrder.PutUint16(b, uint16(raw))
	case 4:
		fi.tag.byteOrder.PutUint32(b, uint32(raw))
	case 8:
		fi.tag.byteOrder.PutUint64(b, raw)
	}
	return b
}

func getScalar(buf []byte, fi *fieldInfo) (uint64, error) {
	need := scalarSize(fi.kind)
	if len(buf) < need {
		return 0, fmt.Errorf("netlink: scalar data too short: need %d bytes, got %d", need, len(buf))
	}
	switch need {
	case 1:
		return uint64(buf[0]), nil
	case 2:
		return uint64(fi.tag.byteOrder.Uint16(buf)), nil
	case 4:
		return uint64(fi.tag.byteOrder.Uint32(buf)), nil
	case 8:
		return fi.tag.byteOrder.Uint64(buf), nil
	default:
		return 0, nil
	}
}

func setScalar(data []byte, fv reflect.Value, fi *fieldInfo) error {
	raw, err := getScalar(data, fi)
	if err != nil {
		return err
	}
	if fv.Kind() == reflect.Pointer {
		ptr := reflect.New(fv.Type().Elem())
		if isSignedKind(fi.kind) {
			ptr.Elem().SetInt(int64(raw))
		} else {
			ptr.Elem().SetUint(raw)
		}
		fv.Set(ptr)
	} else if isSignedKind(fi.kind) {
		fv.SetInt(int64(raw))
	} else {
		fv.SetUint(raw)
	}
	return nil
}

func isSignedKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func (c *Codec) encodeSubMessage(fv reflect.Value, fi *fieldInfo) ([]Attribute, error) {
	fv, ok := unwrapValue(fv)
	if !ok {
		return nil, nil
	}
	children, err := c.encodeAttrs(fv)
	if err != nil {
		return nil, err
	}
	data, err := MarshalAttributes(children)
	if err != nil {
		return nil, err
	}
	return []Attribute{{Type: Nested | fi.tag.attrIndex, Data: data}}, nil
}

func (c *Codec) decodeSubMessage(data []byte, fv reflect.Value, fi *fieldInfo, selector any) error {
	target, err := c.resolveSubMessage(fi.tag.submsg, selector)
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	targetRv := reflect.ValueOf(target)
	children, err := UnmarshalAttributes(data)
	if err != nil {
		return err
	}
	if err := c.decodeAttrs(children, targetRv.Elem()); err != nil {
		return err
	}
	fv.Set(targetRv)
	return nil
}

// resolveSubMessage looks up the resolver for the given sub-message name,
// calls it with the selector value, and validates the result is a *struct.
func (c *Codec) resolveSubMessage(name string, selector any) (any, error) {
	resolve, ok := c.resolvers[name]
	if !ok {
		return nil, fmt.Errorf("netlink: no resolver registered for sub-message %q", name)
	}
	target, err := resolve(selector)
	if err != nil {
		return nil, fmt.Errorf("netlink: resolve sub-message %q: %w", name, err)
	}
	if target == nil {
		return nil, nil
	}
	rt := reflect.TypeOf(target)
	if rt.Kind() != reflect.Pointer || rt.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("netlink: resolver for sub-message %q must return *struct, got %T", name, target)
	}
	return target, nil
}
