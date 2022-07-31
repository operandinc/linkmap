Notice: This package was built at ~1am in about ~25 minutes. If you're thinking of using this in production, would recommend testing stuff before you do (& submitting any fixes/improvements upstream)!

A simple Go package which parses linkmap files. This is a special file format which we use in the Operand API that we use during index to properly assign links to files within a GitHub repository.

Here's a (dumb) example:
foo/$1.{md,mdx} https://example.com/posts/$1
bar/$1/baz/$2.{html} https://example.com/$1/$2

In this case, a file located at foo/xyz.md (relative to the root of the repository) will be mapped to https://example.com/posts/xyz.md.