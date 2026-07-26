package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Analyzer struct {
	columns          int
	hidden           bool
	skipSignatures   bool
	tabWidth         int
	detectFragmented bool
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
		columns, _ := cmd.Flags().GetInt("columns")
		hidden, _ := cmd.Flags().GetBool("hidden")
		skipSignatures, _ := cmd.Flags().GetBool("skip-signatures")
		tabWidth, _ := cmd.Flags().GetInt("tab-width")
		detectFragmented, _ := cmd.Flags().GetBool("detect-fragmented")

		analyzer := Analyzer{
			columns:          columns,
			hidden:           hidden,
			skipSignatures:   skipSignatures,
			tabWidth:         tabWidth,
			detectFragmented: detectFragmented,
		}

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
			if err != nil { //+gocover:ignore:block should never happen
				return err
			}

			if (stat.Mode() & os.ModeCharDevice) != 0 {
				return errors.New("missing file argument or piped input")
			}
		}

		content, err := io.ReadAll(cmd.InOrStdin())
		if err != nil { //+gocover:ignore:block should never happen
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

func Execute(version string) error { //+gocover:ignore:block delegates execution
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().IntP("columns", "c", 80, "maximum line length")
	rootCmd.Flags().IntP("tab-width", "t", 4, "visual width of a tab character")
	rootCmd.Flags().BoolP("skip-signatures", "s", true,
		"skip function signatures")
	rootCmd.Flags().BoolP("hidden", "H", false,
		"include hidden files and directories")
	rootCmd.Flags().BoolP("detect-fragmented", "F", true,
		"detect lines that could be collapsed into one")
}

func (a *Analyzer) NewAnalysis(filePath string, content []byte) *FileAnalysis {
	return &FileAnalysis{analyzer: a, filePath: filePath, content: content}
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

func (f *FileAnalysis) analyzeSignatures(ignoredLines map[int]bool) {
	if f.analyzer.skipSignatures == false {
		return
	}

	if f.filePath == "" || filepath.Ext(f.filePath) == ".go" {
		f.analyzeGoSignatures(ignoredLines)
	}
}

func (f *FileAnalysis) analyzeGoSignatures(ignoredLines map[int]bool) {
	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, "", f.content, parser.ParseComments)
	if err != nil { //+gocover:ignore:block should never happen
		return
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
		case *ast.TypeSpec:
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
}

func (f *FileAnalysis) detectFragmentedLines(out io.Writer) {
	if f.analyzer.detectFragmented == false {
		return
	}

	if f.filePath != "" {
		var ext string
		ext = filepath.Ext(f.filePath)
		if ext != ".go" {
			return
		}
	}

	fileSet := token.NewFileSet()
	node, err := parser.ParseFile(fileSet, "", f.content, parser.ParseComments)
	if err != nil {
		return
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		var startPos token.Position
		startPos = fileSet.Position(n.Pos())
		var endPos token.Position
		endPos = fileSet.Position(n.End())

		if startPos.Line == endPos.Line {
			return true
		}

		var shouldAnalyze bool
		shouldAnalyze = false

		switch n.(type) {
		case *ast.CallExpr:
			shouldAnalyze = true
		case *ast.CompositeLit:
			shouldAnalyze = true
		case *ast.BinaryExpr:
			shouldAnalyze = true
		case *ast.AssignStmt:
			shouldAnalyze = true
		}

		if shouldAnalyze == false {
			return true
		}

		var buf bytes.Buffer
		var cfg printer.Config
		cfg.Mode = printer.RawFormat

		err = cfg.Fprint(&buf, fileSet, n)
		if err != nil { //+gocover:ignore:block should never happen
			return true
		}

		var linearText string
		linearText = buf.String()

		var virtualWidth int
		virtualWidth = (startPos.Column - 1) * f.analyzer.tabWidth

		var lastSpace bool
		lastSpace = false

		for _, char := range linearText {
			if char == '\n' || char == '\r' || char == '\t' || char == ' ' {
				if lastSpace == false {
					virtualWidth++
					lastSpace = true
				}
			} else {
				virtualWidth++
				lastSpace = false
			}
		}

		if virtualWidth < f.analyzer.columns {
			var msg string
			msg = "Warning: Lines %d-%d contain a fragmented expression " +
				"that fits within %d characters (limit %d)\n"

			var l1 int
			l1 = startPos.Line
			var l2 int
			l2 = endPos.Line
			var vw int
			vw = virtualWidth
			var col int
			col = f.analyzer.columns

			fmt.Fprintf(out, msg, l1, l2, vw, col)
		}

		return true
	})
}

func (f *FileAnalysis) Process(out io.Writer) error {
	if f.isBinary() {
		return nil
	}

	ignoredLines := map[int]bool{}

	f.analyzeSignatures(ignoredLines)
	f.detectFragmentedLines(out)

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

	err := scanner.Err()
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
