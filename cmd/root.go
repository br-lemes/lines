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
	allowMultiline   bool
	allowShortIf     bool
	checkSignatures  bool
	columns          int
	filesWithMatches bool
	hidden           bool
	tabWidth         int
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
		allowMultiline, _ := cmd.Flags().GetBool("allow-multiline")
		allowShortIf, _ := cmd.Flags().GetBool("allow-short-if")
		checkSignatures, _ := cmd.Flags().GetBool("check-signatures")
		columns, _ := cmd.Flags().GetInt("columns")
		filesWithMatches, _ := cmd.Flags().GetBool("files-with-matches")
		hidden, _ := cmd.Flags().GetBool("hidden")
		tabWidth, _ := cmd.Flags().GetInt("tab-width")

		analyzer := Analyzer{
			allowMultiline:   allowMultiline,
			allowShortIf:     allowShortIf,
			checkSignatures:  checkSignatures,
			columns:          columns,
			filesWithMatches: filesWithMatches,
			hidden:           hidden,
			tabWidth:         tabWidth,
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
	rootCmd.Flags().Bool("allow-multiline", false,
		"allow expressions split into multiple lines even if it fits on one")
	rootCmd.Flags().Bool("allow-short-if", false,
		"allow if with a short statement (if init; cond)")
	rootCmd.Flags().Bool("check-signatures", false,
		"enforce line limits on function signatures")
	rootCmd.Flags().IntP("columns", "c", 80, "maximum line length")
	rootCmd.Flags().BoolP("files-with-matches", "l", false,
		"print only names of files with lines exceeding the limit")
	rootCmd.Flags().BoolP("hidden", "H", false,
		"include hidden files and directories")
	rootCmd.Flags().IntP("tab-width", "t", 4, "visual width of a tab character")
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
		if a.filesWithMatches {
			fmt.Fprintln(out, filePath)
		} else {
			fmt.Fprintf(out, "%s:\n", filePath)
			out.Write(buf.Bytes())
			fmt.Fprintln(out)
		}
	}

	return nil
}

func (f *FileAnalysis) analyzeSignatures(ignoredLines map[int]bool) {
	if f.analyzer.checkSignatures {
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

func (f *FileAnalysis) detectShortIfStatements(out io.Writer) {
	if f.analyzer.allowShortIf {
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

	lines := bytes.Split(f.content, []byte("\n"))

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		ifStmt, ok := n.(*ast.IfStmt)
		if ok == false {
			return true
		}

		if ifStmt.Init != nil {
			lineNumber := fileSet.Position(ifStmt.Pos()).Line
			var lineContent string
			if lineNumber > 0 && lineNumber <= len(lines) {
				lineContent = string(bytes.TrimRight(lines[lineNumber-1], "\r"))
			}
			fmt.Fprintf(out, "%d: %s // if with a short statement\n",
				lineNumber, lineContent)
		}

		return true
	})
}

func (f *FileAnalysis) detectMultilineExpressions(out io.Writer) {
	if f.analyzer.allowMultiline {
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
			msg = "Warning: Lines %d-%d contain a multiline expression " +
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
	f.detectShortIfStatements(out)
	f.detectMultilineExpressions(out)

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
