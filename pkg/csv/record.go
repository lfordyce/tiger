package csv

import (
	"fmt"
	"os"
)

//Record provides keyed access to the fields of data records where each field
//of a data record is keyed by the value of the corresponding field in the header record.
type Record interface {
	// Header Return the header of the record.
	Header() []string
	// Get the value of the field specified by the key. Returns the empty string
	// if the field does not exist in the record.
	Get(key string) string
	// Put the value into the field specified by the key.
	Put(key string, value string)
	// AsMap returns the contents of the record as a map. Mutation of the map is not supported.
	AsMap() map[string]string
	// AsSlice returns the contents of the record as a slice. Mutation of the slice is not supported.
	AsSlice() []string
	// PutAll puts all the matching values from the specified record into the receiving record
	PutAll(r Record)
	// SameHeader returns true if the receiver and the specified record have the same header.
	SameHeader(r Record) bool
}

type record struct {
	header []string
	index  map[string]int
	fields []string
	cache  map[string]string
}

type RecordBuilder func(fields []string) Record

// NewRecordBuilder returns a function that can be used to create new Records
// for a CSV stream with the specified header.
//
// This can be used with raw encoding/csv streams in cases where a CSV stream contains
// more than one record type.
func NewRecordBuilder(header []string) RecordBuilder {
	index := NewIndex(header)
	return func(fields []string) Record {
		if len(header) < len(fields) {
			if _, err := fmt.Fprintf(os.Stderr, "invariant violated: [%d]fields=%v, [%d]header=%v\n", len(fields), fields, len(header), header); err != nil {
				panic(err)
			}
		}
		tmp := make([]string, len(header))
		copy(tmp, fields)
		return &record{
			header: header,
			index:  index,
			fields: tmp,
		}
	}
}

func (r *record) Header() []string {
	return r.header
}

// Get Answer the value of the field indexed by the column containing the specified header value.
func (r *record) Get(key string) string {
	x, ok := r.index[key]
	if ok && x < len(r.fields) {
		return r.fields[x]
	}
	return ""
}

// Put puts the specified value into the record at the index determined by the key value.
func (r *record) Put(key string, value string) {
	x, ok := r.index[key]
	if ok && x < cap(r.fields) {
		if x > len(r.fields) {
			r.fields = r.fields[0:x]
		}
		if r.cache != nil {
			r.cache[key] = value
		}
		r.fields[x] = value
	}
}

// AsMap return a map containing a copy of the contents of the record.
func (r *record) AsMap() map[string]string {
	if r.cache != nil {
		return r.cache
	}

	result := make(map[string]string)
	for i, h := range r.header {
		if i < len(r.fields) {
			result[h] = r.fields[i]
		} else {
			result[h] = ""
		}
	}
	r.cache = result
	return result
}

// AsSlice return the record values as a slice.
func (r *record) AsSlice() []string {
	return r.fields
}

// PutAll puts all the specified value into the record.
func (r *record) PutAll(o Record) {
	if r.SameHeader(o) {
		copy(r.fields, o.AsSlice())
		r.cache = nil
	} else {
		for i, k := range r.header {
			v := o.Get(k)
			r.fields[i] = v
			if r.cache != nil {
				r.cache[k] = v
			}
		}
	}
}

// SameHeader efficiently check that the receiver and specified records have the same header
func (r *record) SameHeader(o Record) bool {
	h := o.Header()
	if len(r.header) != len(h) {
		return false
	} else if len(h) == 0 || &h[0] == &r.header[0] {
		// two slices with the same address and length have the same contents
		return true
	} else {
		for i, k := range r.header {
			if h[i] != k {
				return false
			}
		}
		return true
	}
}

type Index map[string]int

// NewIndex Return a map that maps each string in the input slice to its index in the slice.
func NewIndex(a []string) Index {
	index := make(map[string]int)
	for i, v := range a {
		index[v] = i
	}
	return index
}

// Contains Answer true if the index contains the specified string.
func (i Index) Contains(k string) bool {
	_, ok := i[k]
	return ok
}
