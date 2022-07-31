// Package linkmap parses and evaluates linkmap files.
// This allows a program to assign a unique URL to a given file in a repository.
package linkmap

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// A Map is a set of rules which map files to links.
type Map struct {
	rules []tuple[template, template]
}

// Parse parses a linkmap and returns a Map object.
func Parse(reader io.Reader) (*Map, error) {
	buf, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %v", err)
	}
	lines := strings.Split(string(buf), "\n")
	var mappings []tuple[template, template]
	for _, l := range lines {
		if l == "" {
			continue
		}
		sub := strings.Split(l, " ")
		if len(sub) != 2 {
			return nil, fmt.Errorf("linkmap: invalid line %q", l)
		}
		in, err := parseTemplate(sub[0])
		if err != nil {
			return nil, fmt.Errorf("linkmap: failed to parse template %q: %w", sub[0], err)
		}
		out, err := parseTemplate(sub[1])
		if err != nil {
			return nil, fmt.Errorf("linkmap: failed to parse template %q: %w", sub[1], err)
		}
		mappings = append(mappings, tuple[template, template]{first: in, second: out})
	}
	// Important to sort by complexity, i.e. longer first.
	sort.Slice(mappings, func(i, j int) bool {
		return len(mappings[i].first) > len(mappings[j].first)
	})
	return &Map{rules: mappings}, nil
}

// ErrNoMatches is returned when no matches were found.
var ErrNoMatches = errors.New("linkmap: no matches found")

// Evaluate evaluates a file path against the map and returns the link.
// If no link was found, an empty string and ErrNoMatches is returned.
func (m *Map) Evaluate(fpath string) (string, error) {
	for _, r := range m.rules {
		variables, didMatch := r.first.match(fpath)
		if !didMatch {
			continue
		}
		link, err := r.second.apply(variables)
		if err != nil {
			return "", fmt.Errorf("failed to apply template: %w", err)
		}
		return link, nil
	}
	return "", ErrNoMatches
}

type tuple[T, E any] struct {
	first  T
	second E
}

type segmentType int

const (
	segmentTypeString segmentType = iota
	segmentTypeVariable
	segmentTypeExtension
)

type segment struct {
	typ segmentType
	val string
}

type template []segment

func parseTemplate(s string) (template, error) {
	var (
		b   strings.Builder
		t   []segment
		ltt segmentType = segmentTypeString
	)
	for _, r := range s {
		switch r {
		case '$':
			if b.Len() > 0 {
				if ltt == segmentTypeVariable {
					return nil, errors.New("linkmap: found two consecutive variables")
				}
				t = append(t, segment{
					typ: ltt,
					val: b.String(),
				})
				b.Reset()
			}
			ltt = segmentTypeVariable
			b.WriteRune(r)
		case '{':
			if b.Len() > 0 {
				t = append(t, segment{
					typ: ltt,
					val: b.String(),
				})
				b.Reset()
			}
			ltt = segmentTypeExtension
			b.WriteRune(r)
		case '}':
			b.WriteRune(r)
			if b.Len() > 0 {
				t = append(t, segment{
					typ: ltt,
					val: b.String(),
				})
				b.Reset()
			}
			ltt = segmentTypeString
		default:
			if ltt == segmentTypeVariable && (r < '0' || r > '9') {
				if b.Len() > 0 {
					t = append(t, segment{
						typ: ltt,
						val: b.String(),
					})
					b.Reset()
				} else {
					return nil, errors.New("linkmap: found variable without preceding number")
				}
				ltt = segmentTypeString
			}
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		t = append(t, segment{
			typ: ltt,
			val: b.String(),
		})
	}
	return t, nil
}

func (tmpl template) equals(other template) bool {
	if len(tmpl) != len(other) {
		return false
	}
	for i := range tmpl {
		if tmpl[i].typ != other[i].typ || tmpl[i].val != other[i].val {
			return false
		}
	}
	return true
}

func (tmpl template) match(s string) (map[string]string, bool) {
	variables := make(map[string]string)
	if len(tmpl) == 0 {
		return variables, s == ""
	}
	var offset int
outer:
	for i, t := range tmpl {
		switch t.typ {
		case segmentTypeString:
			if !strings.HasPrefix(s[offset:], t.val) {
				return nil, false
			}
			offset += len(t.val)
		case segmentTypeExtension:
			possible := strings.Split(t.val[1:len(t.val)-1], ",")
			for _, ext := range possible {
				if strings.HasSuffix(s[offset:], ext) {
					offset += len(ext)
					continue outer
				}
			}
			return nil, false
		case segmentTypeVariable:
			val := s[offset:]
			if i < len(tmpl)-1 {
				next := tmpl[i+1]
				if next.typ == segmentTypeString {
					index := strings.Index(val, next.val)
					if index == -1 {
						return nil, false
					}
					val = val[:index]
				} else if next.typ == segmentTypeExtension {
					possible := strings.Split(next.val[1:len(next.val)-1], ",")
					var index int
					for _, ext := range possible {
						index = strings.Index(val, ext)
						if index != -1 {
							break
						}
					}
					if index == -1 {
						return nil, false
					}
					val = val[:index]
				}
			}
			variables[t.val] = val
			offset += len(val)
		default:
			panic("unexpected link token type")
		}
	}
	return variables, offset == len(s)
}

func (tmpl template) apply(variables map[string]string) (string, error) {
	var b strings.Builder
	for _, t := range tmpl {
		switch t.typ {
		case segmentTypeString:
			b.WriteString(t.val)
		case segmentTypeVariable:
			if val, ok := variables[t.val]; ok {
				b.WriteString(val)
			} else {
				return "", fmt.Errorf("missing variable %s", t.val)
			}
		case segmentTypeExtension:
			return "", fmt.Errorf("extensions not supported")
		default:
			return "", fmt.Errorf("unexpected link token type")
		}
	}
	return b.String(), nil
}
