package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"

	"github.com/br-lemes/lines/internal/version"
	"github.com/spf13/cobra"
)

var (
	columns        int
	skipSignatures bool
	tabWidth       int
)

type Analyzer struct {
	columns        int
	skipSignatures bool
	tabWidth       int
}

var rootCmd = &cobra.Command{
	Use:   "lines [file]",
	Short: "Check file lines that exceed a specific width",
	Long: `Check file lines that exceed a specific width

Arguments:
  [file]   The path to the source file to analyze (optional if using stdin)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var content []byte
		var err error

		if len(args) == 1 {
			content, err = os.ReadFile(args[0])
			if err != nil {
				return err
			}
		} else {
			if cmd.InOrStdin() == os.Stdin {
				stat, err := os.Stdin.Stat()
				if err != nil {
					return err
				}

				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return errors.New("missing file argument or piped input")
				}
			}

			content, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
		}

		analyzer := NewAnalyzer(columns, skipSignatures, tabWidth)
		err = analyzer.Process(content, cmd.OutOrStdout())
		if err != nil {
			return err
		}

		return nil
	},
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.Version = version.GetVersion()
	rootCmd.Flags().IntVarP(&columns, "columns", "c", 80,
		"maximum line length")
	rootCmd.Flags().IntVarP(&tabWidth, "tab-width", "t", 4,
		"visual width of a tab character")
	rootCmd.Flags().BoolVarP(&skipSignatures, "skip-signatures", "s", true,
		"skip function signatures")
}

func NewAnalyzer(cols int, skipSig bool, tabW int) *Analyzer {
	return &Analyzer{
		columns:        cols,
		skipSignatures: skipSig,
		tabWidth:       tabW,
	}
}

func (a *Analyzer) analyzeGoSignatures(content []byte, ignoredLines map[int]bool) error {
	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, "", content, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			startLine := fileSet.Position(x.Pos()).Line
			endLine := fileSet.Position(x.Type.End()).Line
			for i := startLine; i <= endLine; i++ {
				ignoredLines[i] = true
			}
		case *ast.FuncLit:
			startLine := fileSet.Position(x.Pos()).Line
			endLine := fileSet.Position(x.Type.End()).Line
			for i := startLine; i <= endLine; i++ {
				ignoredLines[i] = true
			}
		case *ast.Field:
			_, isFunc := x.Type.(*ast.FuncType)
			if isFunc {
				startLine := fileSet.Position(x.Pos()).Line
				endLine := fileSet.Position(x.End()).Line
				for i := startLine; i <= endLine; i++ {
					ignoredLines[i] = true
				}
			}
		}
		return true
	})

	return nil
}

func (a *Analyzer) Process(content []byte, out io.Writer) error {
	ignoredLines := map[int]bool{}

	if a.skipSignatures {
		err := a.analyzeGoSignatures(content, ignoredLines)
		if err != nil {
			return err
		}
	}

	reader := bytes.NewReader(content)
	scanner := bufio.NewScanner(reader)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		if ignoredLines[lineNumber] {
			continue
		}

		lineWidth := 0
		for _, char := range line {
			if char == '\t' {
				lineWidth += a.tabWidth
			} else {
				lineWidth++
			}
		}

		if lineWidth > a.columns {
			fmt.Fprintf(out, "%d: %s\n", lineNumber, line)
		}
	}

	err := scanner.Err()
	if err != nil {
		return err
	}

	return nil
}
