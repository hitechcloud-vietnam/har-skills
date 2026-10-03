package har

import (
	"encoding/json"
	"strings"
)

// CustomFields stores custom extension fields allowed by the HAR specification that are prefixed with "_".
// The HAR specification permits any field name beginning with "_" as custom extension data,
// such as Chrome's "_initiator", "_priority", and "_resourceType".
type CustomFields map[string]interface{}

// GetCustomField returns the value of a custom extension field.
// It returns nil if the field does not exist.
func (cf CustomFields) GetCustomField(name string) interface{} {
	if cf == nil {
		return nil
	}
	return cf[name]
}

// SetCustomField sets the value of a custom extension field.
// Field names should begin with "_" per the HAR specification, but this is not enforced.
func (cf CustomFields) SetCustomField(name string, value interface{}) {
	if cf == nil {
		return
	}
	cf[name] = value
}

// HasCustomField reports whether the specified custom extension field exists.
func (cf CustomFields) HasCustomField(name string) bool {
	if cf == nil {
		return false
	}
	_, ok := cf[name]
	return ok
}

// DeleteCustomField deletes the specified custom extension field.
func (cf CustomFields) DeleteCustomField(name string) {
	if cf == nil {
		return
	}
	delete(cf, name)
}

// CustomFieldsKeys returns the names of all custom extension fields.
func (cf CustomFields) CustomFieldsKeys() []string {
	if cf == nil {
		return nil
	}
	keys := make([]string, 0, len(cf))
	for k := range cf {
		keys = append(keys, k)
	}
	return keys
}

// knownUnderscoreKeys tracks the _-prefixed JSON keys that are already
// handled as typed struct fields, so they are not duplicated in CustomFields.
var knownUnderscoreKeys = map[string]map[string]bool{
	"Response": {"_transferSize": true, "_error": true},
	"Timings":  {"_blocked_queueing": true, "_blocked_proxy": true},
	"Entries":  {"_initiator": true, "_priority": true, "_resourceType": true},
}

// extractCustomFields extracts custom extension fields prefixed with "_" from raw JSON,
// excluding known extension fields handled by struct fields.
func extractCustomFields(data []byte, typeName string) CustomFields {
	if len(data) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	known := knownUnderscoreKeys[typeName]
	cf := make(CustomFields)
	for key, value := range raw {
		if strings.HasPrefix(key, "_") {
			// Skip keys already handled by struct fields
			if known != nil && known[key] {
				continue
			}
			var v interface{}
			if err := json.Unmarshal(value, &v); err != nil {
				cf[key] = string(value)
			} else {
				cf[key] = v
			}
		}
	}

	if len(cf) == 0 {
		return nil
	}
	return cf
}

// mergeCustomFieldsIntoJSON merges custom extension fields into standard JSON output.
func mergeCustomFieldsIntoJSON(stdData []byte, cf CustomFields) ([]byte, error) {
	if len(cf) == 0 {
		return stdData, nil
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(stdData, &result); err != nil {
		return nil, WrapJSONUnmarshalError(err)
	}

	for key, value := range cf {
		v, err := json.Marshal(value)
		if err != nil {
			return nil, NewJSONParseError("JSON serialization failed", err)
		}
		result[key] = v
	}

	// Keys in result come from valid JSON, and values are successful json.Marshal
	// results (json.RawMessages), so the final Marshal cannot fail.
	data, _ := json.Marshal(result)
	return data, nil
}

// --- Har ---

// GetCustomField returns the custom extension field value for Har.
func (h *Har) GetCustomField(name string) interface{} {
	if h == nil {
		return nil
	}
	return h.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Har.
func (h *Har) SetCustomField(name string, value interface{}) {
	if h == nil {
		return
	}
	if h.CustomFields == nil {
		h.CustomFields = make(CustomFields)
	}
	h.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (h *Har) UnmarshalJSON(data []byte) error {
	if h == nil {
		return NewInvalidFormatError("HAR object is nil")
	}

	type Alias Har
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(h),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	h.CustomFields = extractCustomFields(data, "Har")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
// A value receiver ensures json.Marshal(h) can call this method for either a value or a pointer.
func (h Har) MarshalJSON() ([]byte, error) {
	type Alias Har
	data, err := json.Marshal(Alias(h))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, h.CustomFields)
}

// --- Log ---

// GetCustomField returns the custom extension field value for Log.
func (l *Log) GetCustomField(name string) interface{} {
	if l == nil {
		return nil
	}
	return l.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Log.
func (l *Log) SetCustomField(name string, value interface{}) {
	if l == nil {
		return
	}
	if l.CustomFields == nil {
		l.CustomFields = make(CustomFields)
	}
	l.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (l *Log) UnmarshalJSON(data []byte) error {
	if l == nil {
		return NewInvalidFormatError("Log object is nil")
	}

	type Alias Log
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(l),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	l.CustomFields = extractCustomFields(data, "Log")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (l Log) MarshalJSON() ([]byte, error) {
	type Alias Log
	data, err := json.Marshal(Alias(l))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, l.CustomFields)
}

// --- Entries ---

// GetCustomField returns the custom extension field value for Entries.
func (e *Entries) GetCustomField(name string) interface{} {
	if e == nil {
		return nil
	}
	return e.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Entries.
func (e *Entries) SetCustomField(name string, value interface{}) {
	if e == nil {
		return
	}
	if e.CustomFields == nil {
		e.CustomFields = make(CustomFields)
	}
	e.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (e *Entries) UnmarshalJSON(data []byte) error {
	if e == nil {
		return NewInvalidFormatError("Entries object is nil")
	}

	type Alias Entries
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	e.CustomFields = extractCustomFields(data, "Entries")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (e Entries) MarshalJSON() ([]byte, error) {
	type Alias Entries
	data, err := json.Marshal(Alias(e))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, e.CustomFields)
}

// --- Request ---

// GetCustomField returns the custom extension field value for Request.
func (r *Request) GetCustomField(name string) interface{} {
	if r == nil {
		return nil
	}
	return r.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Request.
func (r *Request) SetCustomField(name string, value interface{}) {
	if r == nil {
		return
	}
	if r.CustomFields == nil {
		r.CustomFields = make(CustomFields)
	}
	r.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (r *Request) UnmarshalJSON(data []byte) error {
	if r == nil {
		return NewInvalidFormatError("Request object is nil")
	}

	type Alias Request
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	r.CustomFields = extractCustomFields(data, "Request")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (r Request) MarshalJSON() ([]byte, error) {
	type Alias Request
	data, err := json.Marshal(Alias(r))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, r.CustomFields)
}

// --- Response ---

// GetCustomField returns the custom extension field value for Response.
func (r *Response) GetCustomField(name string) interface{} {
	if r == nil {
		return nil
	}
	return r.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Response.
func (r *Response) SetCustomField(name string, value interface{}) {
	if r == nil {
		return
	}
	if r.CustomFields == nil {
		r.CustomFields = make(CustomFields)
	}
	r.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
// Response already has _transferSize and _error struct fields, so CustomFields stores only other extensions.
func (r *Response) UnmarshalJSON(data []byte) error {
	if r == nil {
		return NewInvalidFormatError("Response object is nil")
	}

	type Alias Response
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	r.CustomFields = extractCustomFields(data, "Response")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (r Response) MarshalJSON() ([]byte, error) {
	type Alias Response
	data, err := json.Marshal(Alias(r))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, r.CustomFields)
}

// --- Content ---

// GetCustomField returns the custom extension field value for Content.
func (c *Content) GetCustomField(name string) interface{} {
	if c == nil {
		return nil
	}
	return c.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Content.
func (c *Content) SetCustomField(name string, value interface{}) {
	if c == nil {
		return
	}
	if c.CustomFields == nil {
		c.CustomFields = make(CustomFields)
	}
	c.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (c *Content) UnmarshalJSON(data []byte) error {
	if c == nil {
		return NewInvalidFormatError("Content object is nil")
	}

	type Alias Content
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	c.CustomFields = extractCustomFields(data, "Content")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (c Content) MarshalJSON() ([]byte, error) {
	type Alias Content
	// Content contains only int/string fields (CustomFields has json:"-"), so
	// serializing the alias cannot fail.
	data, _ := json.Marshal(Alias(c))
	return mergeCustomFieldsIntoJSON(data, c.CustomFields)
}

// --- Cookie ---

// GetCustomField returns the custom extension field value for Cookie.
func (c *Cookie) GetCustomField(name string) interface{} {
	if c == nil {
		return nil
	}
	return c.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Cookie.
func (c *Cookie) SetCustomField(name string, value interface{}) {
	if c == nil {
		return
	}
	if c.CustomFields == nil {
		c.CustomFields = make(CustomFields)
	}
	c.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (c *Cookie) UnmarshalJSON(data []byte) error {
	if c == nil {
		return NewInvalidFormatError("Cookie object is nil")
	}

	type Alias Cookie
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	c.CustomFields = extractCustomFields(data, "Cookie")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (c Cookie) MarshalJSON() ([]byte, error) {
	type Alias Cookie
	data, err := json.Marshal(Alias(c))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, c.CustomFields)
}

// --- Pages ---

// GetCustomField returns the custom extension field value for Pages.
func (p *Pages) GetCustomField(name string) interface{} {
	if p == nil {
		return nil
	}
	return p.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Pages.
func (p *Pages) SetCustomField(name string, value interface{}) {
	if p == nil {
		return
	}
	if p.CustomFields == nil {
		p.CustomFields = make(CustomFields)
	}
	p.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (p *Pages) UnmarshalJSON(data []byte) error {
	if p == nil {
		return NewInvalidFormatError("Pages object is nil")
	}

	type Alias Pages
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	p.CustomFields = extractCustomFields(data, "Pages")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (p Pages) MarshalJSON() ([]byte, error) {
	type Alias Pages
	data, err := json.Marshal(Alias(p))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, p.CustomFields)
}

// --- Timings ---

// GetCustomField returns the custom extension field value for Timings.
func (t *Timings) GetCustomField(name string) interface{} {
	if t == nil {
		return nil
	}
	return t.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Timings.
func (t *Timings) SetCustomField(name string, value interface{}) {
	if t == nil {
		return
	}
	if t.CustomFields == nil {
		t.CustomFields = make(CustomFields)
	}
	t.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
// Timings already has _blocked_queueing and _blocked_proxy struct fields.
func (t *Timings) UnmarshalJSON(data []byte) error {
	if t == nil {
		return NewInvalidFormatError("Timings object is nil")
	}

	type Alias Timings
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	t.CustomFields = extractCustomFields(data, "Timings")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (t Timings) MarshalJSON() ([]byte, error) {
	type Alias Timings
	data, err := json.Marshal(Alias(t))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, t.CustomFields)
}

// --- Cache ---

// GetCustomField returns the custom extension field value for Cache.
func (c *Cache) GetCustomField(name string) interface{} {
	if c == nil {
		return nil
	}
	return c.CustomFields.GetCustomField(name)
}

// SetCustomField sets a custom extension field value on Cache.
func (c *Cache) SetCustomField(name string, value interface{}) {
	if c == nil {
		return
	}
	if c.CustomFields == nil {
		c.CustomFields = make(CustomFields)
	}
	c.CustomFields.SetCustomField(name, value)
}

// UnmarshalJSON custom-unmarshals JSON and extracts custom extension fields prefixed with "_".
func (c *Cache) UnmarshalJSON(data []byte) error {
	if c == nil {
		return NewInvalidFormatError("Cache object is nil")
	}

	type Alias Cache
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return WrapJSONUnmarshalError(err)
	}

	c.CustomFields = extractCustomFields(data, "Cache")
	return nil
}

// MarshalJSON custom-marshals JSON and merges custom extension fields into the output.
func (c Cache) MarshalJSON() ([]byte, error) {
	type Alias Cache
	data, err := json.Marshal(Alias(c))
	if err != nil {
		return nil, NewJSONParseError("JSON serialization failed", err)
	}
	return mergeCustomFieldsIntoJSON(data, c.CustomFields)
}
