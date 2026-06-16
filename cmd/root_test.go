package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_Stdin_WithSkipSignatures(t *testing.T) {
	inputCode := `package main

func VeryLongFunctionNameToTestIfSignatureIsSkippedCorrectly(a int, b string) error {
	shortLine := 1
	thisIsAVeryLongLineThatExceedsTheEightyColumnsLimitAndShouldBeFlaggedByTheProgram := 2
	return nil
}
`

	bufIn := bytes.NewBufferString(inputCode)
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	rootCmd.SetIn(bufIn)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"--columns=80", "--skip-signatures=true"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	if strings.Contains(output, "func VeryLongFunctionName") {
		t.Errorf("expected function signature to be skipped, but it was found in output")
	}

	expectedLine := "5: \tthisIsAVeryLongLineThatExceedsTheEightyColumnsLimitAndShouldBeFlaggedByTheProgram := 2"
	if strings.Contains(output, expectedLine) == false {
		t.Errorf("expected output to contain flagged line, got:\n%s", output)
	}
}

func TestRootCmd_Stdin_WithoutSkipSignatures(t *testing.T) {
	rootCmd.Flags().Set("columns", "80")
	rootCmd.Flags().Set("skip-signatures", "false")

	inputCode := `package main

func VeryLongFunctionNameToTestIfSignatureIsFlaggedWhenSkipIsFalse(a int, b string) error {
	return nil
}
`

	bufIn := bytes.NewBufferString(inputCode)
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	rootCmd.SetIn(bufIn)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"--columns=80", "--skip-signatures=false"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := bufOut.String()

	expectedLine := "3: func VeryLongFunctionNameToTestIfSignatureIsFlaggedWhenSkipIsFalse(a int, b string) error {"
	if strings.Contains(output, expectedLine) == false {
		t.Errorf("expected function signature to be flagged, got:\n%s", output)
	}
}
