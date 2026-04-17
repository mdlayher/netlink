package netlink

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mdlayher/netlink/nlenc"
)

func ptr[T any](v T) *T {
	return &v
}

var (
	ne = binary.NativeEndian
	be = binary.BigEndian
	le = binary.LittleEndian
)

func mustMarshal(attrs ...Attribute) []byte {
	b, err := MarshalAttributes(attrs)
	if err != nil {
		panic(err)
	}
	return b
}

func put16(order binary.ByteOrder, v uint16) []byte {
	b := make([]byte, 2)
	order.PutUint16(b, v)
	return b
}

func put32(order binary.ByteOrder, v uint32) []byte {
	b := make([]byte, 4)
	order.PutUint32(b, v)
	return b
}

func put64(order binary.ByteOrder, v uint64) []byte {
	b := make([]byte, 8)
	order.PutUint64(b, v)
	return b
}

// 1-100 are reserved for scalars during testing
type Scalars struct {
	Int8    int8  `netlink:"1"`
	Int8Be  int8  `netlink:"2,be"`
	Int8Le  int8  `netlink:"3,le"`
	Int8Ptr *int8 `netlink:"4"`
	Int8Nil *int8 `netlink:"5"`

	UInt8    uint8  `netlink:"6"`
	UInt8Be  uint8  `netlink:"7,be"`
	UInt8Le  uint8  `netlink:"8,le"`
	UInt8Ptr *uint8 `netlink:"9"`
	UInt8Nil *uint8 `netlink:"10"`

	Int16    int16  `netlink:"11"`
	Int16Be  int16  `netlink:"12,be"`
	Int16Le  int16  `netlink:"13,le"`
	Int16Ptr *int16 `netlink:"14"`
	Int16Nil *int16 `netlink:"15"`

	UInt16    uint16  `netlink:"16"`
	UInt16Be  uint16  `netlink:"17,be"`
	UInt16Le  uint16  `netlink:"18,le"`
	UInt16Ptr *uint16 `netlink:"19"`
	UInt16Nil *uint16 `netlink:"20"`

	Int32    int32  `netlink:"21"`
	Int32Be  int32  `netlink:"22,be"`
	Int32Le  int32  `netlink:"23,le"`
	Int32Ptr *int32 `netlink:"24"`
	Int32Nil *int32 `netlink:"25"`

	UInt32    uint32  `netlink:"26"`
	UInt32Be  uint32  `netlink:"27,be"`
	UInt32Le  uint32  `netlink:"28,le"`
	UInt32Ptr *uint32 `netlink:"29"`
	UInt32Nil *uint32 `netlink:"30"`

	Int64    int64  `netlink:"31"`
	Int64Be  int64  `netlink:"32,be"`
	Int64Le  int64  `netlink:"33,le"`
	Int64Ptr *int64 `netlink:"34"`
	Int64Nil *int64 `netlink:"35"`

	UInt64    uint64  `netlink:"36"`
	UInt64Be  uint64  `netlink:"37,be"`
	UInt64Le  uint64  `netlink:"38,le"`
	UInt64Ptr *uint64 `netlink:"39"`
	UInt64Nil *uint64 `netlink:"40"`
}

var ScalarsAttrs = Scalars{
	Int8:    -1,
	Int8Be:  -1,
	Int8Le:  -1,
	Int8Ptr: ptr(int8(-1)),
	Int8Nil: nil,

	UInt8:    1,
	UInt8Be:  1,
	UInt8Le:  1,
	UInt8Ptr: ptr(uint8(1)),
	UInt8Nil: nil,

	Int16:    -2,
	Int16Be:  -2,
	Int16Le:  -2,
	Int16Ptr: ptr(int16(-2)),
	Int16Nil: nil,

	UInt16:    2,
	UInt16Be:  2,
	UInt16Le:  2,
	UInt16Ptr: ptr(uint16(2)),
	UInt16Nil: nil,

	Int32:    -3,
	Int32Be:  -3,
	Int32Le:  -3,
	Int32Ptr: ptr(int32(-3)),
	Int32Nil: nil,

	UInt32:    3,
	UInt32Be:  3,
	UInt32Le:  3,
	UInt32Ptr: ptr(uint32(3)),
	UInt32Nil: nil,

	Int64:    -4,
	Int64Be:  -4,
	Int64Le:  -4,
	Int64Ptr: ptr(int64(-4)),
	Int64Nil: nil,

	UInt64:    4,
	UInt64Be:  4,
	UInt64Le:  4,
	UInt64Ptr: ptr(uint64(4)),
	UInt64Nil: nil,
}

var ScalarsAttrsEncoded = mustMarshal(
	Attribute{Type: 1, Data: []byte{0xff}},
	Attribute{Type: 2, Data: []byte{0xff}},
	Attribute{Type: 3, Data: []byte{0xff}},
	Attribute{Type: 4, Data: []byte{0xff}},

	Attribute{Type: 6, Data: []byte{1}},
	Attribute{Type: 7, Data: []byte{1}},
	Attribute{Type: 8, Data: []byte{1}},
	Attribute{Type: 9, Data: []byte{1}},

	Attribute{Type: 11, Data: put16(ne, 0xfffe)},
	Attribute{Type: 12, Data: put16(be, 0xfffe)},
	Attribute{Type: 13, Data: put16(le, 0xfffe)},
	Attribute{Type: 14, Data: put16(ne, 0xfffe)},

	Attribute{Type: 16, Data: put16(ne, 2)},
	Attribute{Type: 17, Data: put16(be, 2)},
	Attribute{Type: 18, Data: put16(le, 2)},
	Attribute{Type: 19, Data: put16(ne, 2)},

	Attribute{Type: 21, Data: put32(ne, 0xfffffffd)},
	Attribute{Type: 22, Data: put32(be, 0xfffffffd)},
	Attribute{Type: 23, Data: put32(le, 0xfffffffd)},
	Attribute{Type: 24, Data: put32(ne, 0xfffffffd)},

	Attribute{Type: 26, Data: put32(ne, 3)},
	Attribute{Type: 27, Data: put32(be, 3)},
	Attribute{Type: 28, Data: put32(le, 3)},
	Attribute{Type: 29, Data: put32(ne, 3)},

	Attribute{Type: 31, Data: put64(ne, 0xfffffffffffffffc)},
	Attribute{Type: 32, Data: put64(be, 0xfffffffffffffffc)},
	Attribute{Type: 33, Data: put64(le, 0xfffffffffffffffc)},
	Attribute{Type: 34, Data: put64(ne, 0xfffffffffffffffc)},

	Attribute{Type: 36, Data: put64(ne, 4)},
	Attribute{Type: 37, Data: put64(be, 4)},
	Attribute{Type: 38, Data: put64(le, 4)},
	Attribute{Type: 39, Data: put64(ne, 4)},
)

// 101-200 are reserved for strings during testing
type Strings struct {
	String      string  `netlink:"101"`
	StringPtr   *string `netlink:"102"`
	StringEmpty string  `netlink:"103"`
	StringNil   *string `netlink:"104"`
}

var StringAttrs = Strings{
	String:      "Hello, world!",
	StringPtr:   ptr("Hello, world!"),
	StringEmpty: "",
	StringNil:   nil,
}

var StringAttrsEncoded = mustMarshal(
	Attribute{Type: 101, Data: nlenc.Bytes("Hello, world!")},
	Attribute{Type: 102, Data: nlenc.Bytes("Hello, world!")},
)

// 201-300 are reserved for flags during testing
type Flags struct {
	FlagTrue  bool `netlink:"201"`
	FlagFalse bool `netlink:"202"`
}

var FlagAttrs = Flags{
	FlagTrue:  true,
	FlagFalse: false,
}

var FlagAttrsEncoded = mustMarshal(
	Attribute{Type: 201, Data: []byte{}},
)

// 301-400 are reserved for multi attributes during testing
type SliceMulti struct {
	Int8Multi      []int8 `netlink:"301,multi"`
	Int8MultiEmpty []int8 `netlink:"302,multi"`

	UInt8Multi      []uint8 `netlink:"303,multi"`
	UInt8MultiEmpty []uint8 `netlink:"304,multi"`

	Int16Multi      []int16 `netlink:"305,multi"`
	Int16MultiEmpty []int16 `netlink:"306,multi"`

	UInt16Multi      []uint16 `netlink:"307,multi"`
	UInt16MultiEmpty []uint16 `netlink:"308,multi"`

	Int32Multi      []int32 `netlink:"309,multi"`
	Int32MultiEmpty []int32 `netlink:"310,multi"`

	UInt32Multi      []uint32 `netlink:"311,multi"`
	UInt32MultiEmpty []uint32 `netlink:"312,multi"`

	Int64Multi      []int64 `netlink:"313,multi"`
	Int64MultiEmpty []int64 `netlink:"314,multi"`

	UInt64Multi      []uint64 `netlink:"315,multi"`
	UInt64MultiEmpty []uint64 `netlink:"316,multi"`

	StringMulti      []string `netlink:"317,multi"`
	StringMultiEmpty []string `netlink:"318,multi"`
}

var SliceMultiAttrs = SliceMulti{
	Int8Multi: []int8{1, 2, 3},

	UInt8Multi: []uint8{1, 2, 3},

	Int16Multi: []int16{1, 2, 3},

	UInt16Multi: []uint16{1, 2, 3},

	Int32Multi: []int32{1, 2, 3},

	UInt32Multi: []uint32{1, 2, 3},

	Int64Multi: []int64{1, 2, 3},

	UInt64Multi: []uint64{1, 2, 3},

	StringMulti: []string{"foo", "bar"},
}

var SliceMultiAttrsEncoded = mustMarshal(
	Attribute{Type: 301, Data: []byte{1}},
	Attribute{Type: 301, Data: []byte{2}},
	Attribute{Type: 301, Data: []byte{3}},

	Attribute{Type: 303, Data: []byte{1}},
	Attribute{Type: 303, Data: []byte{2}},
	Attribute{Type: 303, Data: []byte{3}},

	Attribute{Type: 305, Data: put16(ne, 1)},
	Attribute{Type: 305, Data: put16(ne, 2)},
	Attribute{Type: 305, Data: put16(ne, 3)},

	Attribute{Type: 307, Data: put16(ne, 1)},
	Attribute{Type: 307, Data: put16(ne, 2)},
	Attribute{Type: 307, Data: put16(ne, 3)},

	Attribute{Type: 309, Data: put32(ne, 1)},
	Attribute{Type: 309, Data: put32(ne, 2)},
	Attribute{Type: 309, Data: put32(ne, 3)},

	Attribute{Type: 311, Data: put32(ne, 1)},
	Attribute{Type: 311, Data: put32(ne, 2)},
	Attribute{Type: 311, Data: put32(ne, 3)},

	Attribute{Type: 313, Data: put64(ne, 1)},
	Attribute{Type: 313, Data: put64(ne, 2)},
	Attribute{Type: 313, Data: put64(ne, 3)},

	Attribute{Type: 315, Data: put64(ne, 1)},
	Attribute{Type: 315, Data: put64(ne, 2)},
	Attribute{Type: 315, Data: put64(ne, 3)},

	Attribute{Type: 317, Data: nlenc.Bytes("foo")},
	Attribute{Type: 317, Data: nlenc.Bytes("bar")},
)

// 401-500 are reserved for indexed arrays during testing
type SliceIndexed struct {
	IndexedInt8      []uint8 `netlink:"401,indexed"`
	IndexedInt8Empty []uint8 `netlink:"402,indexed"`

	IndexedUInt8      []uint8 `netlink:"403,indexed"`
	IndexedUInt8Empty []uint8 `netlink:"404,indexed"`

	IndexedInt16      []uint16 `netlink:"405,indexed"`
	IndexedInt16Empty []uint16 `netlink:"406,indexed"`

	IndexedUInt16      []uint16 `netlink:"407,indexed"`
	IndexedUInt16Empty []uint16 `netlink:"408,indexed"`

	IndexedInt32      []uint32 `netlink:"409,indexed"`
	IndexedInt32Empty []uint32 `netlink:"410,indexed"`

	IndexedUInt32      []uint32 `netlink:"411,indexed"`
	IndexedUInt32Empty []uint32 `netlink:"412,indexed"`

	IndexedInt64      []uint64 `netlink:"413,indexed"`
	IndexedInt64Empty []uint64 `netlink:"414,indexed"`

	IndexedUInt64      []uint64 `netlink:"415,indexed"`
	IndexedUInt64Empty []uint64 `netlink:"416,indexed"`

	IndexedString      []string `netlink:"417,indexed"`
	IndexedStringEmpty []string `netlink:"418,indexed"`
}

var SliceIndexedAttrs = SliceIndexed{
	IndexedInt8:   []uint8{1, 2, 3},
	IndexedUInt8:  []uint8{1, 2, 3},
	IndexedInt16:  []uint16{1, 2, 3},
	IndexedUInt16: []uint16{1, 2, 3},
	IndexedInt32:  []uint32{1, 2, 3},
	IndexedUInt32: []uint32{1, 2, 3},
	IndexedInt64:  []uint64{1, 2, 3},
	IndexedUInt64: []uint64{1, 2, 3},
	IndexedString: []string{"foo", "bar"},
}

var IndexedArrayAttrsEncoded = mustMarshal(
	Attribute{Type: Nested | 401, Data: mustMarshal(
		Attribute{Type: 0, Data: []byte{1}},
		Attribute{Type: 1, Data: []byte{2}},
		Attribute{Type: 2, Data: []byte{3}},
	)},
	Attribute{Type: Nested | 403, Data: mustMarshal(
		Attribute{Type: 0, Data: []byte{1}},
		Attribute{Type: 1, Data: []byte{2}},
		Attribute{Type: 2, Data: []byte{3}},
	)},
	Attribute{Type: Nested | 405, Data: mustMarshal(
		Attribute{Type: 0, Data: put16(ne, 1)},
		Attribute{Type: 1, Data: put16(ne, 2)},
		Attribute{Type: 2, Data: put16(ne, 3)},
	)},
	Attribute{Type: Nested | 407, Data: mustMarshal(
		Attribute{Type: 0, Data: put16(ne, 1)},
		Attribute{Type: 1, Data: put16(ne, 2)},
		Attribute{Type: 2, Data: put16(ne, 3)},
	)},
	Attribute{Type: Nested | 409, Data: mustMarshal(
		Attribute{Type: 0, Data: put32(ne, 1)},
		Attribute{Type: 1, Data: put32(ne, 2)},
		Attribute{Type: 2, Data: put32(ne, 3)},
	)},
	Attribute{Type: Nested | 411, Data: mustMarshal(
		Attribute{Type: 0, Data: put32(ne, 1)},
		Attribute{Type: 1, Data: put32(ne, 2)},
		Attribute{Type: 2, Data: put32(ne, 3)},
	)},
	Attribute{Type: Nested | 413, Data: mustMarshal(
		Attribute{Type: 0, Data: put64(ne, 1)},
		Attribute{Type: 1, Data: put64(ne, 2)},
		Attribute{Type: 2, Data: put64(ne, 3)},
	)},
	Attribute{Type: Nested | 415, Data: mustMarshal(
		Attribute{Type: 0, Data: put64(ne, 1)},
		Attribute{Type: 1, Data: put64(ne, 2)},
		Attribute{Type: 2, Data: put64(ne, 3)},
	)},
	Attribute{Type: Nested | 417, Data: mustMarshal(
		Attribute{Type: 0, Data: nlenc.Bytes("foo")},
		Attribute{Type: 1, Data: nlenc.Bytes("bar")},
	)},
)

// 601-700 are reserved for binary attributes during testing
type Binary struct {
	Data    []byte `netlink:"601"`
	DataNil []byte `netlink:"602"`
}

var BinaryAttrs = Binary{
	Data: []byte{0xde, 0xad, 0xbe, 0xef},
}

var BinaryAttrsEncoded = mustMarshal(
	Attribute{Type: 601, Data: []byte{0xde, 0xad, 0xbe, 0xef}},
)

// NestItem is a simple struct used to test multi-nested and indexed-nested.
type NestItem struct {
	Value uint32 `netlink:"1001"`
	Name  string `netlink:"1002"`
}

// 701-800 are reserved for multi nested attributes during testing
type MultiNest struct {
	Items      []NestItem `netlink:"701"`
	ItemsEmpty []NestItem `netlink:"702"`
}

var MultiNestAttrs = MultiNest{
	Items: []NestItem{
		{Value: 1, Name: "foo"},
		{Value: 2, Name: "bar"},
	},
}

var MultiNestAttrsEncoded = mustMarshal(
	Attribute{Type: Nested | 701, Data: mustMarshal(
		Attribute{Type: 1001, Data: put32(ne, 1)},
		Attribute{Type: 1002, Data: nlenc.Bytes("foo")},
	)},
	Attribute{Type: Nested | 701, Data: mustMarshal(
		Attribute{Type: 1001, Data: put32(ne, 2)},
		Attribute{Type: 1002, Data: nlenc.Bytes("bar")},
	)},
)

// 801-900 are reserved for indexed nested attributes during testing
type IndexedNest struct {
	Items      []NestItem `netlink:"801,indexed"`
	ItemsEmpty []NestItem `netlink:"802,indexed"`
}

var IndexedNestAttrs = IndexedNest{
	Items: []NestItem{
		{Value: 1, Name: "foo"},
		{Value: 2, Name: "bar"},
	},
}

var IndexedNestAttrsEncoded = mustMarshal(
	Attribute{Type: Nested | 801, Data: mustMarshal(
		Attribute{Type: Nested, Data: mustMarshal(
			Attribute{Type: 1001, Data: put32(ne, 1)},
			Attribute{Type: 1002, Data: nlenc.Bytes("foo")},
		)},
		Attribute{Type: Nested | 1, Data: mustMarshal(
			Attribute{Type: 1001, Data: put32(ne, 2)},
			Attribute{Type: 1002, Data: nlenc.Bytes("bar")},
		)},
	)},
)

// 901-1000 are reserved for sub-message testing
type SubMsgContainer struct {
	Kind string     `netlink:"901"`
	Data SubMsgData `netlink:"902,submsg=test-submsg,selector=Kind"`
}

// SubMsgData is a sealed interface: only FormatA and FormatB satisfy it.
type SubMsgData interface {
	isSubMsgData()
}

type FormatA struct {
	X uint32 `netlink:"903"`
	Y uint32 `netlink:"904"`
}

func (*FormatA) isSubMsgData() {}

type FormatB struct {
	Label string `netlink:"905"`
}

func (*FormatB) isSubMsgData() {}

var testResolver = map[string]SubMessageResolver{
	"test-submsg": func(selector any) (any, error) {
		switch selector.(string) {
		case "format-a":
			return &FormatA{}, nil
		case "format-b":
			return &FormatB{}, nil
		}
		return nil, fmt.Errorf("unknown test-submsg selector: %q", selector)
	},
}

var testCodec = NewAttributeCodec(&AttributeCodecConfig{Resolvers: testResolver})

var SubMsgContainerA = SubMsgContainer{
	Kind: "format-a",
	Data: &FormatA{X: 10, Y: 20},
}

var SubMsgContainerAEncoded = mustMarshal(
	Attribute{Type: 901, Data: nlenc.Bytes("format-a")},
	Attribute{Type: Nested | 902, Data: mustMarshal(
		Attribute{Type: 903, Data: put32(ne, 10)},
		Attribute{Type: 904, Data: put32(ne, 20)},
	)},
)

var SubMsgContainerB = SubMsgContainer{
	Kind: "format-b",
	Data: &FormatB{Label: "hello"},
}

var SubMsgContainerBEncoded = mustMarshal(
	Attribute{Type: 901, Data: nlenc.Bytes("format-b")},
	Attribute{Type: Nested | 902, Data: mustMarshal(
		Attribute{Type: 905, Data: nlenc.Bytes("hello")},
	)},
)

var SubMsgContainerNil = SubMsgContainer{
	Kind: "unknown",
	Data: nil,
}

var SubMsgContainerNilEncoded = mustMarshal(
	Attribute{Type: 901, Data: nlenc.Bytes("unknown")},
)

// 501-600 are reserved for nested attributes during testing
type NestedStruct struct {
	Scalars    Scalars  `netlink:"501"`
	ScalarsPtr *Scalars `netlink:"502"`
	ScalarsNil *Scalars `netlink:"503"`

	Strings    Strings  `netlink:"504"`
	StringsPtr *Strings `netlink:"505"`
	StringsNil *Strings `netlink:"506"`

	Flags    Flags  `netlink:"507"`
	FlagsPtr *Flags `netlink:"508"`
	FlagsNil *Flags `netlink:"509"`

	Multi    SliceMulti  `netlink:"510"`
	MultiPtr *SliceMulti `netlink:"511"`
	MultiNil *SliceMulti `netlink:"512"`

	Indexed    SliceIndexed  `netlink:"513"`
	IndexedPtr *SliceIndexed `netlink:"514"`
	IndexedNil *SliceIndexed `netlink:"515"`
}

var NestedStructAttrs = NestedStruct{
	Scalars:    ScalarsAttrs,
	ScalarsPtr: &ScalarsAttrs,
	Strings:    StringAttrs,
	StringsPtr: &StringAttrs,
	Flags:      FlagAttrs,
	FlagsPtr:   &FlagAttrs,
	Multi:      SliceMultiAttrs,
	MultiPtr:   &SliceMultiAttrs,
	Indexed:    SliceIndexedAttrs,
	IndexedPtr: &SliceIndexedAttrs,
}

var NestedStructEncoded = mustMarshal(
	Attribute{Type: Nested | 501, Data: ScalarsAttrsEncoded},
	Attribute{Type: Nested | 502, Data: ScalarsAttrsEncoded},
	Attribute{Type: Nested | 504, Data: StringAttrsEncoded},
	Attribute{Type: Nested | 505, Data: StringAttrsEncoded},
	Attribute{Type: Nested | 507, Data: FlagAttrsEncoded},
	Attribute{Type: Nested | 508, Data: FlagAttrsEncoded},
	Attribute{Type: Nested | 510, Data: SliceMultiAttrsEncoded},
	Attribute{Type: Nested | 511, Data: SliceMultiAttrsEncoded},
	Attribute{Type: Nested | 513, Data: IndexedArrayAttrsEncoded},
	Attribute{Type: Nested | 514, Data: IndexedArrayAttrsEncoded},
)

type Mixed struct {
	ExportedAttribute   string    `netlink:"1101"`
	unexportedAttribute string    `netlink:"1102"`
	SlicePtr            []*string `netlink:"1103"`
	ExportedField       string
	unexportedField     string
}

var MixedAttrs = Mixed{
	ExportedAttribute:   "exported-attr",
	unexportedAttribute: "unexported-attr",
	SlicePtr:            []*string{ptr("slice-ptr")},
	ExportedField:       "exported-field",
	unexportedField:     "unexported-field",
}

var MixedAttrsEncoded = mustMarshal(
	Attribute{Type: 1101, Data: nlenc.Bytes("exported-attr")},
	Attribute{Type: 1103, Data: nlenc.Bytes("slice-ptr")},
)

func TestMarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []byte
		wantErr bool
	}{
		{
			name:  "scalars",
			input: ScalarsAttrs,
			want:  ScalarsAttrsEncoded,
		},
		{
			name:  "string",
			input: StringAttrs,
			want:  StringAttrsEncoded,
		},
		{
			name:  "flag",
			input: FlagAttrs,
			want:  FlagAttrsEncoded,
		},
		{
			name:  "multi",
			input: SliceMultiAttrs,
			want:  SliceMultiAttrsEncoded,
		},
		{
			name:  "indexed-array",
			input: SliceIndexedAttrs,
			want:  IndexedArrayAttrsEncoded,
		},
		{
			name:  "nested attributes",
			input: NestedStructAttrs,
			want:  NestedStructEncoded,
		},
		{
			name:  "binary",
			input: BinaryAttrs,
			want:  BinaryAttrsEncoded,
		},
		{
			name:  "multi-nest",
			input: MultiNestAttrs,
			want:  MultiNestAttrsEncoded,
		},
		{
			name:  "indexed-nest",
			input: IndexedNestAttrs,
			want:  IndexedNestAttrsEncoded,
		},
		{
			name:  "submessage-a",
			input: SubMsgContainerA,
			want:  SubMsgContainerAEncoded,
		},
		{
			name:  "submessage-b",
			input: SubMsgContainerB,
			want:  SubMsgContainerBEncoded,
		},
		{
			name:  "submessage-nil",
			input: SubMsgContainerNil,
			want:  SubMsgContainerNilEncoded,
		},
		{
			name:  "mixed",
			input: MixedAttrs,
			want:  MixedAttrsEncoded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testCodec.Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Marshal() = diff: %s", diff)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		got     any
		want    any
		wantErr bool
	}{
		{
			name:  "scalars",
			input: ScalarsAttrsEncoded,
			got:   &Scalars{},
			want:  &ScalarsAttrs,
		},
		{
			name:  "string",
			input: StringAttrsEncoded,
			got:   &Strings{},
			want:  &StringAttrs,
		},
		{
			name:  "flag",
			input: FlagAttrsEncoded,
			got:   &Flags{},
			want:  &FlagAttrs,
		},
		{
			name:  "slice multi",
			input: SliceMultiAttrsEncoded,
			got:   &SliceMulti{},
			want:  &SliceMultiAttrs,
		},
		{
			name:  "slice indexed",
			input: IndexedArrayAttrsEncoded,
			got:   &SliceIndexed{},
			want:  &SliceIndexedAttrs,
		},
		{
			name:  "nested attributes",
			input: NestedStructEncoded,
			got:   &NestedStruct{},
			want:  &NestedStructAttrs,
		},
		{
			name:  "binary",
			input: BinaryAttrsEncoded,
			got:   &Binary{},
			want:  &BinaryAttrs,
		},
		{
			name:  "multi-nest",
			input: MultiNestAttrsEncoded,
			got:   &MultiNest{},
			want:  &MultiNestAttrs,
		},
		{
			name:  "indexed-nest",
			input: IndexedNestAttrsEncoded,
			got:   &IndexedNest{},
			want:  &IndexedNestAttrs,
		},
		{
			name:  "mixed",
			input: MixedAttrsEncoded,
			got:   &Mixed{},
			want: &Mixed{
				ExportedAttribute: "exported-attr",
				SlicePtr:          []*string{ptr("slice-ptr")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.got
			err := testCodec.Unmarshal(tt.input, got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.IgnoreUnexported(Mixed{})); diff != "" {
				t.Errorf("Unmarshal() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestMarshalSubMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []byte
		wantErr bool
	}{
		{
			name:  "submessage-a",
			input: SubMsgContainerA,
			want:  SubMsgContainerAEncoded,
		},
		{
			name:  "submessage-b",
			input: SubMsgContainerB,
			want:  SubMsgContainerBEncoded,
		},
		{
			name:  "submessage-nil",
			input: SubMsgContainerNil,
			want:  SubMsgContainerNilEncoded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testCodec.Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Marshal() = diff: %s", diff)
			}
		})
	}
}

func TestUnmarshalSubMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		got     any
		want    any
		wantErr bool
	}{
		{
			name:  "submessage-a",
			input: SubMsgContainerAEncoded,
			got:   &SubMsgContainer{},
			want:  &SubMsgContainerA,
		},
		{
			name:  "submessage-b",
			input: SubMsgContainerBEncoded,
			got:   &SubMsgContainer{},
			want:  &SubMsgContainerB,
		},
		{
			name:  "submessage-nil",
			input: SubMsgContainerNilEncoded,
			got:   &SubMsgContainer{},
			want:  &SubMsgContainerNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.got
			err := testCodec.Unmarshal(tt.input, got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Unmarshal() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
