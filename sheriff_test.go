package sheriff

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type AModel struct {
	AllGroups bool `json:"something" groups:"test"`
	TestGroup bool `json:"something_else" groups:"test-other"`
}

type TestGroupsModel struct {
	DefaultMarshal            string            `json:"default_marshal"`
	NeverMarshal              string            `json:"-"`
	OnlyGroupTest             string            `json:"only_group_test" groups:"test"`
	OnlyGroupTestNeverMarshal string            `json:"-" groups:"test"`
	OnlyGroupTestOther        string            `json:"only_group_test_other" groups:"test-other"`
	GroupTestAndOther         string            `json:"group_test_and_other" groups:"test,test-other"`
	OmitEmpty                 string            `json:"omit_empty,omitempty"`
	OmitEmptyGroupTest        string            `json:"omit_empty_group_test,omitempty" groups:"test"`
	SliceString               []string          `json:"slice_string,omitempty" groups:"test"`
	MapStringStruct           map[string]AModel `json:"map_string_struct,omitempty" groups:"test,test-other"`
}

func TestMarshal_GroupsValidGroup(t *testing.T) {
	testModel := &TestGroupsModel{
		DefaultMarshal:     "DefaultMarshal",
		NeverMarshal:       "NeverMarshal",
		OnlyGroupTest:      "OnlyGroupTest",
		OnlyGroupTestOther: "OnlyGroupTestOther",
		GroupTestAndOther:  "GroupTestAndOther",
		OmitEmpty:          "OmitEmpty",
		OmitEmptyGroupTest: "OmitEmptyGroupTest",
		SliceString:        []string{"test", "bla"},
		MapStringStruct:    map[string]AModel{"firstModel": {true, true}},
	}

	o := &Options{
		Groups: []string{"test"},
	}

	actualMap, err := Marshal(o, testModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"default_marshal":       "DefaultMarshal",
		"only_group_test":       "OnlyGroupTest",
		"omit_empty":            "OmitEmpty",
		"omit_empty_group_test": "OmitEmptyGroupTest",
		"group_test_and_other":  "GroupTestAndOther",
		"map_string_struct": map[string]map[string]bool{
			"firstModel": {
				"something": true,
			},
		},
		"slice_string": []string{"test", "bla"},
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestMarshal_GroupsValidGroupOmitEmpty(t *testing.T) {
	testModel := &TestGroupsModel{
		DefaultMarshal:     "DefaultMarshal",
		NeverMarshal:       "NeverMarshal",
		OnlyGroupTest:      "OnlyGroupTest",
		OnlyGroupTestOther: "OnlyGroupTestOther",
		GroupTestAndOther:  "GroupTestAndOther",
		OmitEmpty:          "OmitEmpty",
		SliceString:        []string{"test", "bla"},
		MapStringStruct:    map[string]AModel{"firstModel": {true, true}},
	}

	o := &Options{
		Groups: []string{"test"},
	}

	actualMap, err := Marshal(o, testModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"default_marshal":      "DefaultMarshal",
		"only_group_test":      "OnlyGroupTest",
		"group_test_and_other": "GroupTestAndOther",
		"omit_empty":           "OmitEmpty",
		"map_string_struct": map[string]map[string]bool{
			"firstModel": {
				"something": true,
			},
		},
		"slice_string": []string{"test", "bla"},
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestMarshal_GroupsInvalidGroup(t *testing.T) {
	testModel := &TestGroupsModel{
		DefaultMarshal:     "DefaultMarshal",
		NeverMarshal:       "NeverMarshal",
		OnlyGroupTest:      "OnlyGroupTest",
		OnlyGroupTestOther: "OnlyGroupTestOther",
		GroupTestAndOther:  "GroupTestAndOther",
		OmitEmpty:          "OmitEmpty",
		OmitEmptyGroupTest: "OmitEmptyGroupTest",
		MapStringStruct:    map[string]AModel{"firstModel": {true, true}},
	}

	o := &Options{
		Groups: []string{"foo"},
	}

	actualMap, err := Marshal(o, testModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]string{
		"default_marshal": "DefaultMarshal",
		"omit_empty":      "OmitEmpty"})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestMarshal_GroupsNoGroups(t *testing.T) {
	testModel := &TestGroupsModel{
		DefaultMarshal:     "DefaultMarshal",
		NeverMarshal:       "NeverMarshal",
		OnlyGroupTest:      "OnlyGroupTest",
		OnlyGroupTestOther: "OnlyGroupTestOther",
		GroupTestAndOther:  "GroupTestAndOther",
		OmitEmpty:          "OmitEmpty",
		OmitEmptyGroupTest: "OmitEmptyGroupTest",
		MapStringStruct:    map[string]AModel{"firstModel": {true, true}},
	}

	o := &Options{}

	actualMap, err := Marshal(o, testModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"default_marshal": "DefaultMarshal",
		"omit_empty":      "OmitEmpty",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type IsMarshaller struct {
	ShouldMarshal string `json:"should_marshal" groups:"test"`
}

func (i IsMarshaller) Marshal(options *Options) (interface{}, error) {
	return Marshal(options, i)
}

type TestRecursiveModel struct {
	SomeData     string             `json:"some_data" groups:"test"`
	GroupsData   []*TestGroupsModel `json:"groups_data,omitempty" groups:"test"`
	IsMarshaller IsMarshaller       `json:"is_marshaller" groups:"test"`
}

func TestMarshal_Recursive(t *testing.T) {
	testModel := &TestGroupsModel{
		DefaultMarshal:     "DefaultMarshal",
		NeverMarshal:       "NeverMarshal",
		OnlyGroupTest:      "OnlyGroupTest",
		OnlyGroupTestOther: "OnlyGroupTestOther",
		GroupTestAndOther:  "GroupTestAndOther",
		OmitEmpty:          "OmitEmpty",
		OmitEmptyGroupTest: "OmitEmptyGroupTest",
		SliceString:        []string{"test", "bla"},
		MapStringStruct:    map[string]AModel{"firstModel": {true, true}},
	}
	testRecursiveModel := &TestRecursiveModel{
		SomeData:     "SomeData",
		GroupsData:   []*TestGroupsModel{testModel},
		IsMarshaller: IsMarshaller{"test"},
	}

	o := &Options{
		Groups: []string{"test"},
	}

	actualMap, err := Marshal(o, testRecursiveModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"some_data": "SomeData",
		"groups_data": []map[string]interface{}{
			{
				"default_marshal":       "DefaultMarshal",
				"only_group_test":       "OnlyGroupTest",
				"omit_empty_group_test": "OmitEmptyGroupTest",
				"group_test_and_other":  "GroupTestAndOther",
				"omit_empty":            "OmitEmpty",
				"map_string_struct": map[string]map[string]bool{
					"firstModel": {
						"something": true,
					},
				},
				"slice_string": []string{"test", "bla"},
			},
		},
		"is_marshaller": map[string]interface{}{
			"should_marshal": "test",
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type TestNoJSONTagModel struct {
	SomeData    string `groups:"test"`
	AnotherData string `groups:"test"`
}

func TestMarshal_NoJSONTAG(t *testing.T) {
	testModel := &TestNoJSONTagModel{
		SomeData:    "SomeData",
		AnotherData: "AnotherData",
	}

	o := &Options{
		Groups: []string{"test"},
	}

	actualMap, err := Marshal(o, testModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"SomeData":    "SomeData",
		"AnotherData": "AnotherData",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type UserInfo struct {
	UserPrivateInfo `groups:"private"`
	UserPublicInfo  `groups:"public"`
}
type UserPrivateInfo struct {
	Age string
}
type UserPublicInfo struct {
	ID    string
	Email string `groups:"private"`
}

func TestMarshal_ParentInherit(t *testing.T) {
	publicInfo := UserPublicInfo{ID: "F94", Email: "hello@hello.com"}
	privateInfo := UserPrivateInfo{Age: "20"}
	testModel := UserInfo{
		UserPrivateInfo: privateInfo,
		UserPublicInfo:  publicInfo,
	}

	o := &Options{
		Groups: []string{"public"},
	}

	actualMap, err := Marshal(o, testModel)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"ID": "F94",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))

}

type TimeHackTest struct {
	ATime time.Time `json:"a_time" groups:"test"`
}

func TestMarshal_TimeHack(t *testing.T) {
	hackCreationTime, err := time.Parse(time.RFC3339, "2017-01-20T18:11:00Z")
	assert.NoError(t, err)

	tht := TimeHackTest{
		ATime: hackCreationTime,
	}
	o := &Options{
		Groups: []string{"test"},
	}

	actualMap, err := Marshal(o, tht)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"a_time": "2017-01-20T18:11:00Z",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type EmptyMapTest struct {
	AMap map[string]string `json:"a_map" groups:"test"`
}

func TestMarshal_EmptyMap(t *testing.T) {
	emp := EmptyMapTest{
		AMap: make(map[string]string),
	}
	o := &Options{
		Groups: []string{"test"},
	}

	actualMap, err := Marshal(o, emp)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"a_map": nil,
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type TestMarshal_Embedded struct {
	Foo string `json:"foo" groups:"test"`
}

type TestMarshal_EmbeddedParent struct {
	*TestMarshal_Embedded
	Bar string `json:"bar" groups:"test"`
}

func TestMarshal_EmbeddedField(t *testing.T) {
	v := TestMarshal_EmbeddedParent{
		&TestMarshal_Embedded{"Hello"},
		"World",
	}
	o := &Options{Groups: []string{"test"}}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"bar": "World",
		"foo": "Hello",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type TestMarshal_EmbeddedEmpty struct {
	Foo string `groups:"nothing"`
}

type TestMarshal_EmbeddedParentEmpty struct {
	*TestMarshal_EmbeddedEmpty
	Bar string `json:"bar" groups:"test"`
}

func TestMarshal_EmbeddedFieldEmpty(t *testing.T) {
	v := TestMarshal_EmbeddedParentEmpty{
		&TestMarshal_EmbeddedEmpty{"Hello"},
		"World",
	}
	o := &Options{Groups: []string{"test"}}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"bar": "World",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type InterfaceableBeta struct {
	Integer int    `json:"integer" groups:"safe"`
	Secret  string `json:"secret" groups:"unsafe"`
}
type InterfaceableCharlie struct {
	Integer int    `json:"integer" groups:"safe"`
	Secret  string `json:"secret" groups:"unsafe"`
}
type ArrayOfInterfaceable []CanHazInterface
type CanHazInterface interface {
}
type InterfacerAlpha struct {
	Plaintext     string               `json:"plaintext" groups:"safe"`
	Secret        string               `json:"secret" groups:"unsafe"`
	Nested        InterfaceableBeta    `json:"nested" groups:"safe"`
	Interfaceable ArrayOfInterfaceable `json:"interfaceable" groups:"safe"`
}

func TestMarshal_ArrayOfInterfaceable(t *testing.T) {
	a := InterfacerAlpha{
		"I am plaintext",
		"I am a secret",
		InterfaceableBeta{
			100,
			"Still a secret",
		},
		ArrayOfInterfaceable{
			InterfaceableBeta{200, "Still a secret good"},
			InterfaceableCharlie{300, "Still a secret excellent"},
		}}

	o := &Options{
		Groups: []string{"safe"},
	}

	actualMap, err := Marshal(o, a)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"interfaceable": []map[string]interface{}{
			map[string]interface{}{"integer": 200},
			map[string]interface{}{"integer": 300},
		},
		"nested":    map[string]interface{}{"integer": 100},
		"plaintext": "I am plaintext",
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type TestInlineStruct struct {
	// explicitely testing unexported fields
	// go vet complains about it and that's ok to ignore.
	tableName        struct{ Test string } `json:"-" is:"notexported"`
	tableNameWithTag struct{ Test string } `json:"foo" is:"notexported"`

	Field  string  `json:"field"`
	Field2 *string `json:"field2"`
}

func TestMarshal_InlineStruct(t *testing.T) {
	v := TestInlineStruct{
		tableName:        struct{ Test string }{"test"},
		tableNameWithTag: struct{ Test string }{"testWithTag"},
		Field:            "World",
		Field2:           nil,
	}
	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"field":  "World",
		"field2": nil,
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type TestInet struct {
	IPv4 net.IP `json:"ipv4"`
	IPv6 net.IP `json:"ipv6"`
}

func TestMarshal_Inet(t *testing.T) {
	v := TestInet{
		IPv4: net.ParseIP("0.0.0.0").To4(),
		IPv6: net.ParseIP("::").To16(),
	}
	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"ipv4": net.ParseIP("0.0.0.0").To4(),
		"ipv6": net.ParseIP("::").To16(),
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type TestArray struct {
	Foo []int `json:"foo" groups:"summary"`
	Bar int   `json:"bar"`
}

func TestArrayEmpty(t *testing.T) {
	v := TestArray{
		Foo: []int{3, 1, 4},
		Bar: 0,
	}

	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"bar": 0,
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))

}

type TopLevel struct {
	NestedAnon
	Named NestedNamed `json:"named"`
}

type NestedAnon struct {
	Foo int `json:"foo"`
	Bar int `json:"bar"`
}

type NestedNamed struct {
	Name string `json:"name" groups:"verbose"`
}

func TestAnonymousWithNoGroups(t *testing.T) {
	anon := NestedAnon{
		Foo: 3,
		Bar: 4,
	}

	named := NestedNamed{
		Name: "KooKoo",
	}
	v := TopLevel{anon, named}

	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"foo":   3,
		"bar":   4,
		"named": map[string]interface{}{},
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type MapAliasContainer struct {
	Foo MapAlias `json:"foo"`
}

type MapAlias map[string]bool

func TestEmptyMapAlias(t *testing.T) {
	v := MapAliasContainer{
		Foo: MapAlias{},
	}
	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"foo": MapAlias{},
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestNilMapAlias(t *testing.T) {
	v := MapAliasContainer{
		Foo: nil,
	}
	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"foo": nil,
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

type ArrayAliasContainer struct {
	Foo ArrayAlias `json:"foo"`
}

type ArrayAlias []int

func TestEmptyArrayAlias(t *testing.T) {
	v := ArrayAliasContainer{
		Foo: ArrayAlias{},
	}
	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"foo": ArrayAlias{},
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestNilArrayAlias(t *testing.T) {
	v := ArrayAliasContainer{
		Foo: nil,
	}
	o := &Options{}

	actualMap, err := Marshal(o, v)
	assert.NoError(t, err)

	actual, err := json.Marshal(actualMap)
	assert.NoError(t, err)

	expected, err := json.Marshal(map[string]interface{}{
		"foo": nil,
	})
	assert.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}
