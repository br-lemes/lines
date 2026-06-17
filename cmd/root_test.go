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
				`16: 	fmt.Printf("short: %d\nlong: %d\n", short, long)`,
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
				"6: 	Handler func(payload string, retryCount int) error",
				"10: 	ProcessEvent(id int, data []byte, force bool) bool",
				"13: func ExecuteSignatureCheck() {",
				`16: 	fmt.Printf("short: %d\nlong: %d\n", short, long)`,
				"22: 	callback := func(status int, message string) {",
			},
			unexpectedLines: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bufOut := new(bytes.Buffer)
			rootCmd.SetOut(bufOut)
			rootCmd.SetArgs(tt.args)

			if err := rootCmd.Execute(); err != nil {
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

	if err := rootCmd.Execute(); err != nil {
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
