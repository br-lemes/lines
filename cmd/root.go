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
	"path/filepath"

	"github.com/br-lemes/lines/internal/version"
	"github.com/spf13/cobra"
)

var (
	columns        int
	hidden         bool
	skipSignatures bool
	tabWidth       int
)

type Analyzer struct {
	columns        int
	hidden         bool
	skipSignatures bool
	tabWidth       int
}

type FileAnalysis struct {
	analyzer *Analyzer
	filePath string
	content  []byte
}

var rootCmd = &cobra.Command{
	Use:   "lines [file...]",
	Short: "Check file lines that exceed a specific width",
	Long: `Check file lines that exceed a specific width

Arguments:
  [file...]   The paths to the source files or directories`,
	RunE: func(cmd *cobra.Command, args []string) error {
		analyzer := NewAnalyzer(columns, hidden, skipSignatures, tabWidth)

		if len(args) > 0 {
			for _, filePath := range args {
				info, err := os.Lstat(filePath)
				if err != nil {
					return err
				}

				if (info.Mode() & os.ModeSymlink) != 0 {
					continue
				}

				if info.IsDir() {
					err = analyzer.ProcessDir(filePath, cmd.OutOrStdout())
				} else {
					err = analyzer.ProcessFile(filePath, cmd.OutOrStdout())
				}

				if err != nil {
					return err
				}
			}
			return nil
		}

		if cmd.InOrStdin() == os.Stdin {
			stat, err := os.Stdin.Stat()
			if err != nil {
				return err
			}

			if (stat.Mode() & os.ModeCharDevice) != 0 {
				return errors.New("missing file argument or piped input")
			}
		}

		content, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return err
		}

		analysis := analyzer.NewAnalysis("", content)
		err = analysis.Process(cmd.OutOrStdout())
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
	rootCmd.Flags().BoolVarP(&hidden, "hidden", "H", false,
		"include hidden files and directories")
}

func NewAnalyzer(cols int, hid bool, skipSig bool, tabW int) *Analyzer {
	return &Analyzer{
		columns:        cols,
		hidden:         hid,
		skipSignatures: skipSig,
		tabWidth:       tabW,
	}
}

func (a *Analyzer) NewAnalysis(filePath string, content []byte) *FileAnalysis {
	return &FileAnalysis{
		analyzer: a,
		filePath: filePath,
		content:  content,
	}
}

func (a *Analyzer) ProcessDir(dirPath string, out io.Writer) error {
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if (d.Type() & os.ModeSymlink) != 0 {
			return nil
		}

		name := d.Name()
		if a.hidden == false {
			if len(name) > 1 {
				if name[0] == '.' {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		if d.IsDir() == false {
			fileErr := a.ProcessFile(path, out)
			if fileErr != nil {
				return fileErr
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (a *Analyzer) ProcessFile(filePath string, out io.Writer) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	analysis := a.NewAnalysis(filePath, content)

	buf := new(bytes.Buffer)
	err = analysis.Process(buf)
	if err != nil {
		return err
	}

	if buf.Len() > 0 {
		fmt.Fprintf(out, "%s:\n", filePath)
		out.Write(buf.Bytes())
		fmt.Fprintln(out)
	}

	return nil
}

func (f *FileAnalysis) analyzeSignatures(ignoredLines map[int]bool) error {
	if f.analyzer.skipSignatures == false {
		return nil
	}

	if f.filePath == "" || filepath.Ext(f.filePath) == ".go" {
		return f.analyzeGoSignatures(ignoredLines)
	}

	return nil
}

func (f *FileAnalysis) analyzeGoSignatures(ignoredLines map[int]bool) error {
	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, "", f.content, parser.ParseComments)
	if err != nil {
		return nil
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

func (f *FileAnalysis) Process(out io.Writer) error {
	if f.isBinary() {
		return nil
	}

	ignoredLines := map[int]bool{}

	err := f.analyzeSignatures(ignoredLines)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(f.content)
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
				lineWidth += f.analyzer.tabWidth
			} else {
				lineWidth++
			}
		}

		if lineWidth > f.analyzer.columns {
			fmt.Fprintf(out, "%d: %s\n", lineNumber, line)
		}
	}

	err = scanner.Err()
	if err != nil {
		return err
	}

	return nil
}

func (f *FileAnalysis) isBinary() bool {
	upperBound := 8192
	if len(f.content) < upperBound {
		upperBound = len(f.content)
	}

	for i := 0; i < upperBound; i++ {
		if f.content[i] == 0x00 {
			return true
		}
	}
	return false
}
