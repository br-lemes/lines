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
- **Pipeline Friendly**: Seamlessly accepts input from either a file argument or
directly via standard input (`stdin`).
