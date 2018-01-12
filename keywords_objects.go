package jsonschema

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// MaxProperties MUST be a non-negative integer.
// An object instance is valid against "maxProperties" if its number of properties is less than, or equal to, the value of this keyword.
type MaxProperties int

// Validate implements the validator interface for MaxProperties
func (m MaxProperties) Validate(data interface{}) error {
	if obj, ok := data.(map[string]interface{}); ok {
		if len(obj) > int(m) {
			return fmt.Errorf("%d object properties exceed %d maximum", len(obj), m)
		}
	}
	return nil
}

// MinProperties MUST be a non-negative integer.
// An object instance is valid against "minProperties" if its number of properties is greater than, or equal to, the value of this keyword.
// Omitting this keyword has the same behavior as a value of 0.
type MinProperties int

// Validate implements the validator interface for MinProperties
func (m MinProperties) Validate(data interface{}) error {
	if obj, ok := data.(map[string]interface{}); ok {
		if len(obj) < int(m) {
			return fmt.Errorf("%d object properties below %d minimum", len(obj), m)
		}
	}
	return nil
}

// Required ensures that for a given object instance, every item in the array is the name of a property in the instance.
// The value of this keyword MUST be an array. Elements of this array, if any, MUST be strings, and MUST be unique.
// Omitting this keyword has the same behavior as an empty array.
type Required []string

// Validate implements the validator interface for Required
func (r Required) Validate(data interface{}) error {
	if obj, ok := data.(map[string]interface{}); ok {
		for _, key := range r {
			if val, ok := obj[key]; val == nil && !ok {
				return fmt.Errorf(`"%s" value is required`, key)
			}
		}
	}
	return nil
}

// Properties MUST be an object. Each value of this object MUST be a valid JSON Schema.
// This keyword determines how child instances validate for objects, and does not directly validate
// the immediate instance itself.
// Validation succeeds if, for each name that appears in both the instance and as a name within this
// keyword's value, the child instance for that name successfully validates against the corresponding schema.
// Omitting this keyword has the same behavior as an empty object.
type Properties map[string]*Schema

// Validate implements the validator interface for Properties
func (p Properties) Validate(data interface{}) error {
	if obj, ok := data.(map[string]interface{}); ok {
		for key, val := range obj {
			if p[key] != nil {
				if err := p[key].Validate(val); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// PatternProperties determines how child instances validate for objects, and does not directly validate the immediate instance itself.
// Validation of the primitive instance type against this keyword always succeeds.
// Validation succeeds if, for each instance name that matches any regular expressions that appear as a property name in this
// keyword's value, the child instance for that name successfully validates against each schema that corresponds to a matching
// regular expression.
// Each property name of this object SHOULD be a valid regular expression,
// according to the ECMA 262 regular expression dialect.
// Each property value of this object MUST be a valid JSON Schema.
// Omitting this keyword has the same behavior as an empty object.
type PatternProperties []patternSchema

type patternSchema struct {
	key    string
	re     *regexp.Regexp
	schema *Schema
}

// Validate implements the validator interface for PatternProperties
func (p PatternProperties) Validate(data interface{}) error {
	if obj, ok := data.(map[string]interface{}); ok {
		for key, val := range obj {
			for _, ptn := range p {
				if ptn.re.Match([]byte(key)) {
					if err := ptn.schema.Validate(val); err != nil {
						return fmt.Errorf("object key %s pattern prop %s error: %s", key, ptn.key, err.Error())
					}
				}
			}
		}
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for PatternProperties
func (p *PatternProperties) UnmarshalJSON(data []byte) error {
	var props map[string]*Schema
	if err := json.Unmarshal(data, &props); err != nil {
		return err
	}

	ptn := make(PatternProperties, len(props))
	i := 0
	for key, sch := range props {
		re, err := regexp.Compile(key)
		if err != nil {
			return fmt.Errorf("invalid pattern: %s: %s", key, err.Error())
		}
		ptn[i] = patternSchema{
			key:    key,
			re:     re,
			schema: sch,
		}
		i++
	}

	*p = ptn
	return nil
}

// AdditionalProperties determines how child instances validate for objects, and does not directly validate the immediate instance itself.
// Validation with "additionalProperties" applies only to the child values of instance names that do not match any names in "properties",
// and do not match any regular expression in "patternProperties".
// For all such properties, validation succeeds if the child instance validates against the "additionalProperties" schema.
// Omitting this keyword has the same behavior as an empty schema.
type AdditionalProperties struct {
	properties Properties
	patterns   PatternProperties
	Schema     Schema
}

// Validate implements the validator interface for AdditionalProperties
func (ap AdditionalProperties) Validate(data interface{}) error {
	if obj, ok := data.(map[string]interface{}); ok {
	KEYS:
		for key, val := range obj {
			for propKey := range ap.properties {
				if propKey == key {
					continue KEYS
				}
			}
			for _, ptn := range ap.patterns {
				if ptn.re.Match([]byte(key)) {
					continue KEYS
				}
			}
			if err := ap.Schema.Validate(val); err != nil {
				return fmt.Errorf("object key %s additionalProperties error: %s", key, err.Error())
			}
		}
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for AdditionalProperties
func (ap *AdditionalProperties) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*ap = AdditionalProperties{Schema: sch}
	return nil
}

// Dependencies : [CREF1]
// This keyword specifies rules that are evaluated if the instance is an object and contains a
// certain property.
// This keyword's value MUST be an object. Each property specifies a dependency.
// Each dependency value MUST be an array or a valid JSON Schema.
// If the dependency value is a subschema, and the dependency key is a property in the instance,
// the entire instance must validate against the dependency value.
// If the dependency value is an array, each element in the array, if any, MUST be a string,
// and MUST be unique. If the dependency key is a property in the instance, each of the items
// in the dependency value must be a property that exists in the instance.
// Omitting this keyword has the same behavior as an empty object.
type Dependencies map[string][]*Schema

// Validate implements the validator interface for Dependencies
func (d Dependencies) Validate(data interface{}) error {
	return nil
}

// PropertyNames checks if every property name in the instance validates against the provided schema
// if the instance is an object.
// Note the property name that the schema is testing will always be a string.
// Omitting this keyword has the same behavior as an empty schema.
type PropertyNames Schema

// Validate implements the validator interface for PropertyNames
func (p PropertyNames) Validate(data interface{}) error {
	sch := Schema(p)
	if obj, ok := data.(map[string]interface{}); ok {
		for key := range obj {
			if err := sch.Validate(key); err != nil {
				return fmt.Errorf("invalid propertyName: %s", err.Error())
			}
		}
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for PropertyNames
func (p *PropertyNames) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*p = PropertyNames(sch)
	return nil
}