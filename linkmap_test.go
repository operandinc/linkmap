package linkmap

import "testing"

func TestTokenizeLink(t *testing.T) {
	cases := []struct {
		link   string
		expect []segment
	}{
		{
			link: "foo/posts/$1.{md,mdx}",
			expect: []segment{
				{
					typ: segmentTypeString,
					val: "foo/posts/",
				},
				{
					typ: segmentTypeVariable,
					val: "$1",
				},
				{
					typ: segmentTypeString,
					val: ".",
				},
				{
					typ: segmentTypeExtension,
					val: "{md,mdx}",
				},
			},
		},
		{
			link: "foo/$1/bar/$2.{html}",
			expect: []segment{
				{
					typ: segmentTypeString,
					val: "foo/",
				},
				{
					typ: segmentTypeVariable,
					val: "$1",
				},
				{
					typ: segmentTypeString,
					val: "/bar/",
				},
				{
					typ: segmentTypeVariable,
					val: "$2",
				},
				{
					typ: segmentTypeString,
					val: ".",
				},
				{
					typ: segmentTypeExtension,
					val: "{html}",
				},
			},
		},
		{
			link: "https://example.com/posts/$1",
			expect: []segment{
				{
					typ: segmentTypeString,
					val: "https://example.com/posts/",
				},
				{
					typ: segmentTypeVariable,
					val: "$1",
				},
			},
		},
	}
	for _, c := range cases {
		if got, err := parseTemplate(c.link); err != nil {
			t.Errorf("parseTemplate(%q) error: %v", c.link, err)
		} else if !got.equals(c.expect) {
			t.Errorf("parseTemplate(%q) = %v; want %v", c.link, got, c.expect)
		}
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		link     string
		retTrue  []string
		retFalse []string
	}{
		{
			link:    "foo/posts/$1.{md,mdx}",
			retTrue: []string{"foo/posts/abc.md", "foo/posts/yyz.mdx"},
			retFalse: []string{
				"foo/posts/abc.html",
				"bar/content/abc.md",
				"foo/posts/abc.mdx.md",
				"foo/postsabc.md",
			},
		},
		{
			link: "https://example.com/posts/$1",
			retTrue: []string{
				"https://example.com/posts/abc",
				"https://example.com/posts/yyz",
				"https://example.com/posts/",
			},
			retFalse: []string{
				"https://example.com/content/abc",
				"http://example.com/posts/abc",
			},
		},
	}
	for _, c := range cases {
		tokenized, err := parseTemplate(c.link)
		if err != nil {
			t.Errorf("parseTemplate(%q) error: %v", c.link, err)
		}
		for _, s := range c.retTrue {
			if _, ok := tokenized.match(s); !ok {
				t.Errorf("match(%q) = false; want true", s)
			}
		}
		for _, s := range c.retFalse {
			if _, ok := tokenized.match(s); ok {
				t.Errorf("match(%q) = true; want false", s)
			}
		}
	}
}

func TestComplete(t *testing.T) {
	cases := []struct {
		inTmpl  string
		outTmpl string
		in      string
		expect  string
	}{
		{
			inTmpl:  "foo/posts/$1.{md,mdx}",
			outTmpl: "https://example.com/posts/$1",
			in:      "foo/posts/abc.md",
			expect:  "https://example.com/posts/abc",
		},
		{
			inTmpl:  "foo/$1/bar/$2.{html}",
			outTmpl: "https://example.com/$1/$2.html",
			in:      "foo/abc/bar/xyz.html",
			expect:  "https://example.com/abc/xyz.html",
		},
	}
	for _, c := range cases {
		inTokenized, err := parseTemplate(c.inTmpl)
		if err != nil {
			t.Errorf("parseTemplate(%q) error: %v", c.inTmpl, err)
		}
		outTokenized, err := parseTemplate(c.outTmpl)
		if err != nil {
			t.Errorf("parseTemplate(%q) error: %v", c.outTmpl, err)
		}

		variables, ok := inTokenized.match(c.in)
		if !ok {
			t.Errorf("match(%q) = false; want true", c.in)
		}

		got, err := outTokenized.apply(variables)
		if err != nil {
			t.Errorf("apply(%q) error: %v", c.in, err)
		}

		if got != c.expect {
			t.Errorf("apply(%q) = %q; want %q", c.in, got, c.expect)
		}
	}
}
