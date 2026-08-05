# lines

`lines` is a lightweight Go command-line tool built with Cobra that scans source
files to find lines exceeding a specific visual width limit. It accurately
handles tab characters and can automatically skip function signatures to prevent
false positives on boilerplate code.

## Features

- **Tab Width Awareness**: Measures tab characters (`\t`) based on their visual
column size rather than a single character byte.
- **AST Smart Filtering**: Leverages Go's native syntax parser to identify and
skip function signatures, anonymous functions, and function-typed struct fields.
Use `--check-signatures` to disable this.
- **Short If Detection**: Flags `if` statements that use a short init statement
(`if init; cond { ... }`), such as `if v := math.Pow(x, n); v < lim {`. This is
disallowed by default, matching the project's preference against combining
variable initialization and condition checks on one line. Use `--allow-short-if`
to disable this check.
- **Multiline Expression Detection**: Identifies expressions (like multi-line
calls, assignments, literals, or binary operations) that were broken down into
multiple lines but could completely fit within a single line limit. Use
`--allow-multiline` to disable this check.
- **Pipeline Friendly**: Seamlessly accepts input from either a file argument or
directly via standard input (`stdin`).
