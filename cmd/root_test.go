package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCmd_SingleFile(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", "testdata/first.txt",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()
	expectedLine := "2: this line exceeds ten characters"
	if strings.Contains(output, expectedLine) == false {
		t.Errorf("expected output to contain flagged line, got:\n%s", output)
	}
}

func TestRootCmd_MultipleFiles(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4",
		"testdata/first.txt", "testdata/second.txt",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "testdata/first.txt:") == false {
		t.Errorf("expected header for first.txt")
	}

	if strings.Contains(output, "testdata/second.txt:") == false {
		t.Errorf("expected header for second.txt")
	}

	expectedLine1 := "2: this line exceeds ten characters"
	expectedLine2 := "2: this is another long line"

	if strings.Contains(output, expectedLine1) == false {
		t.Errorf("expected output to contain first flagged line")
	}

	if strings.Contains(output, expectedLine2) == false {
		t.Errorf("expected output to contain second flagged line")
	}
}

func TestRootCmd_Directory(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", "testdata/dir",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "testdata/dir/inner.txt:") == false {
		t.Errorf("expected header for inner.txt")
	}

	expectedLine := "2: line that is too long"
	if strings.Contains(output, expectedLine) == false {
		t.Errorf("expected output to contain flagged line, got:\n%s", output)
	}
}

func TestRootCmd_Hidden_Disabled(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", "testdata/hidden",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "testdata/hidden/.hidden.txt") ||
		strings.Contains(output, "testdata/hidden/.dir/nested.txt") {
		t.Errorf("expected hidden file to be ignored")
	}

	if strings.Contains(output, "testdata/hidden/normal.txt") == false {
		t.Errorf("expected normal file to be processed")
	}
}

func TestRootCmd_Hidden_Enabled(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=true", "-s=true", "-t=4", "testdata/hidden",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "testdata/hidden/.hidden.txt") == false &&
		strings.Contains(output, "testdata/hidden/.dir/nested.txt") == false {
		t.Errorf("expected hidden file to be processed")
	}

	if strings.Contains(output, "testdata/hidden/normal.txt") == false {
		t.Errorf("expected normal file to be processed")
	}
}

func TestRootCmd_Binary(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", "testdata/binary.dat",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if len(output) > 0 {
		t.Errorf("expected no output for binary file, but got:\n%s", output)
	}
}

func TestRootCmd_GoSignatures(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedLines   []string
		unexpectedLines []string
	}{
		{
			name: "Skip Enabled",
			args: []string{
				"-c=20", "-H=false", "-s=true", "-t=4",
				"testdata/signatures.go",
			},
			expectedLines: []string{
				`18: 	fmt.Printf("short: %d\nlong: %d\n", short, long)`,
			},
			unexpectedLines: []string{
				"Handler func(payload string, retryCount int) error",
				"ProcessEvent(id int, data []byte, force bool) bool",
				"func ExecuteSignatureCheck() {",
				"callback := func(status int, message string, active bool) {",
			},
		},
		{
			name: "Skip Disabled",
			args: []string{
				"-c=20", "-H=false", "-s=false", "-t=4",
				"testdata/signatures.go",
			},
			expectedLines: []string{
				"8: 	Handler func(payload string, retryCount int) error",
				"12: 	ProcessEvent(id int, data []byte, force bool) bool",
				"15: func ExecuteSignatureCheck() {",
				`18: 	fmt.Printf("short: %d\nlong: %d\n", short, long)`,
				"24: 	callback := func(status int, message string) {",
			},
			unexpectedLines: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bufOut := new(bytes.Buffer)
			rootCmd.SetOut(bufOut)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			output := bufOut.String()

			for _, exp := range tt.expectedLines {
				if !strings.Contains(output, exp) {
					t.Errorf("expected output to contain %q, got:\n%s",
						exp, output)
				}
			}

			for _, unexpected := range tt.unexpectedLines {
				if strings.Contains(output, unexpected) {
					t.Errorf("expected output NOT to contain %q, got:\n%s",
						unexpected, output)
				}
			}
		})
	}
}

func TestRootCmd_TabWidth(t *testing.T) {
	tests := []struct {
		name        string
		tabWidth    string
		expectLine2 bool
		errorMsg    string
	}{
		{
			name:        "TabWidth4_Ignored",
			tabWidth:    "4",
			expectLine2: false,
			errorMsg:    "expected tab line to be ignored for tab-width=4",
		},
		{
			name:        "TabWidth8_Flagged",
			tabWidth:    "8",
			expectLine2: true,
			errorMsg:    "expected tab line to be flagged for tab-width=8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bufOut := new(bytes.Buffer)
			rootCmd.SetOut(bufOut)

			rootCmd.SetArgs([]string{
				"-c=10", "-H=false", "-s=true", "-t=" + tt.tabWidth,
				"testdata/tab_width.txt",
			})

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			output := bufOut.String()

			if strings.Contains(output, "1:     12345") {
				t.Errorf("line 1 (spaces) should never exceed 10 characters")
			}

			hasLine2 := strings.Contains(output, "2: \t12345")
			if hasLine2 != tt.expectLine2 {
				t.Errorf("%s, got:\n%s", tt.errorMsg, output)
			}
		})
	}
}

func TestRootCmd_Symlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "symlink_scan_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetDir, err := os.MkdirTemp("", "symlink_target_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(targetDir)

	regularFile := filepath.Join(tmpDir, "regular.txt")
	err = os.WriteFile(regularFile, []byte("this line is very long\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	targetFile := filepath.Join(tmpDir, "target.txt")
	err = os.WriteFile(targetFile, []byte("another very long line\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	symlinkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink(targetFile, symlinkFile)
	if err != nil {
		t.Skipf("symlinks not supported on this OS: %v", err)
	}

	err = os.WriteFile(filepath.Join(targetDir, "hidden_long.txt"),
		[]byte("nested long line inside target\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	symlinkDir := filepath.Join(tmpDir, "link_dir")
	err = os.Symlink(targetDir, symlinkDir)
	if err != nil {
		t.Skipf("symlinks not supported on this OS: %v", err)
	}

	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetArgs([]string{"-c=10", "-H=false", "-s=true", "-t=4", tmpDir})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "regular.txt") == false {
		t.Errorf("expected regular file to be processed")
	}

	if strings.Contains(output, "link.txt") {
		t.Errorf("expected symlink to be skipped, got:\n%s", output)
	}

	if strings.Contains(output, "link_dir") ||
		strings.Contains(output, "hidden_long.txt") {
		t.Errorf("expected directory symlink to be skipped, got:\n%s", output)
	}

	bufOut.Reset()
	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", symlinkFile,
	})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error in direct call, got %v", err)
	}

	if bufOut.Len() > 0 {
		t.Errorf("expected direct symlink argument to be ignored, got:\n%s",
			bufOut.String())
	}
}

func TestRootCmd_NoInputError(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{"-c=10", "-H=false", "-s=true", "-t=4"})

	err := rootCmd.Execute()

	if err == nil {
		t.Fatalf("expected an error when no input file or stdin is provided")
	}

	expected := "missing file argument or piped input"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error message to contain %q, got: %v", expected, err)
	}
}

func TestRootCmd_Stdin(t *testing.T) {
	inputCode := "line\n"

	bufIn := bytes.NewBufferString(inputCode)
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	rootCmd.SetIn(bufIn)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"-c=2", "-H=false", "-s=true", "-t=4"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()
	expectedLine := "1: line"
	if strings.Contains(output, expectedLine) == false {
		t.Errorf("expected output to contain flagged line, got:\n%s", output)
	}
}

func TestRootCmd_DetectFragmented(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetArgs([]string{
		"-c=80", "-H=false", "-s=true", "-t=4", "-F=true",
		"testdata/fragmented.go",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	expectedWarnings := []string{
		"Warning: Lines 11-14 contain a fragmented expression",
		"Warning: Lines 16-19 contain a fragmented expression",
		"Warning: Lines 23-24 contain a fragmented expression",
		"Warning: Lines 26-27 contain a fragmented expression",
	}

	for _, exp := range expectedWarnings {
		contains := strings.Contains(output, exp)
		if contains == false {
			t.Errorf("expected output to contain warning, got:\n%s", output)
		}
	}

	unexpected := "Warning: Lines 29"
	containsUnexpected := strings.Contains(output, unexpected)
	if containsUnexpected == true {
		t.Errorf(
			"expected long expression NOT to trigger warning, but it did:\n%s",
			output)
	}
}

func TestRootCmd_FragmentedDisabled(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=80", "-H=false", "-s=true", "-t=4", "-F=false",
		"testdata/fragmented.go",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()
	if strings.Contains(output, "Warning: Lines") {
		t.Errorf(
			"expected no fragmented warnings when flag is disabled, got:\n%s",
			output)
	}
}

func TestRootCmd_LstatError(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	nonExistentFile := filepath.Join("testdata", "this_file_does_not_exist.go")
	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", nonExistentFile,
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error for non-existent file, got nil")
	}

	if os.IsNotExist(err) == false {
		t.Errorf("expected a not exist error, got: %v", err)
	}
}

func TestRootCmd_WalkDirError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "walk_dir_error_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	badDir := filepath.Join(tmpDir, "unreadable_dir")
	err = os.Mkdir(badDir, 0000)
	if err != nil {
		t.Fatalf("failed to create unreadable dir: %v", err)
	}

	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetArgs([]string{"-c=10", "-H=false", "-s=true", "-t=4", tmpDir})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error when walking unreadable directory, got nil")
	}

	if os.IsPermission(err) == false {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestRootCmd_ReadFilePermissionError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "read_file_err_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "secret.txt")
	err = os.WriteFile(tmpFile, []byte("hidden"), 0000)
	if err != nil {
		t.Fatalf("failed to create unreadable file: %v", err)
	}

	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetArgs([]string{"-c=10", "-H=false", "-s=true", "-t=4", tmpFile})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error for unreadable file, got nil")
	}

	if os.IsPermission(err) == false {
		t.Errorf("expected permission error, got: %v", err)
	}
}

func TestRootCmd_LineTooLongError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scanner_error_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hugeLine := strings.Repeat("A", 70000) + "\n"
	hugeFile := filepath.Join(tmpDir, "huge_line.txt")
	err = os.WriteFile(hugeFile, []byte(hugeLine), 0644)
	if err != nil {
		t.Fatalf("failed to create huge file: %v", err)
	}

	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	stdinBuffer := bytes.NewBufferString(hugeLine)
	rootCmd.SetIn(stdinBuffer)
	rootCmd.SetArgs([]string{"-c=10", "-H=false", "-s=true", "-t=4"})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error from stdin scanner, got nil")
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Errorf("expected 'token too long' error, got: %v", err)
	}

	bufOut.Reset()
	rootCmd.SetOut(bufOut)
	rootCmd.SetArgs([]string{"-c=10", "-H=false", "-s=true", "-t=4", tmpDir})
	err = rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected an error from directory scanner, got nil")
	}
	if !strings.Contains(err.Error(), "token too long") {
		t.Errorf("expected 'token too long' error, got: %v", err)
	}
}

func TestRootCmd_FilesWithMatches(t *testing.T) {
	bufOut := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)

	rootCmd.SetArgs([]string{
		"-c=10", "-H=false", "-s=true", "-t=4", "-l=true",
		"testdata/first.txt", "testdata/second.txt",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "testdata/first.txt") == false {
		t.Errorf("expected output to contain first.txt")
	}

	if strings.Contains(output, "testdata/second.txt") == false {
		t.Errorf("expected output to contain second.txt")
	}

	if strings.Contains(output, "testdata/first.txt:") {
		t.Errorf("expected plain filename without colon header")
	}

	unexpectedLine := "2: this line exceeds ten characters"
	if strings.Contains(output, unexpectedLine) {
		t.Errorf("expected output NOT to contain line contents when -l is set,"+
			" got:\n%s", output)
	}
}
