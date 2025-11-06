package ui

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewDisplay(t *testing.T) {
	display := NewDisplay()

	if display.reader == nil {
		t.Error("Expected reader to be initialized")
	}
}

func TestShowCommand(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	display := NewDisplay()
	display.ShowCommand("ls -la")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "Command: ls -la\n"
	if output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}
}

func TestConfirmExecution_Yes(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"\n", true}, // Empty (just Enter)
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"NO\n", false},
		{"maybe\n", false},
	}

	for _, tc := range testCases {
		reader := bufio.NewReader(strings.NewReader(tc.input))
		display := &Display{reader: reader}

		result, err := display.ConfirmExecution()
		if err != nil {
			t.Errorf("ConfirmExecution() with input '%s' failed: %v", strings.TrimSpace(tc.input), err)
		}

		if result != tc.expected {
			t.Errorf("For input '%s': expected %v, got %v", strings.TrimSpace(tc.input), tc.expected, result)
		}
	}
}

func TestConfirmExecution_EOF(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	display := &Display{reader: reader}

	_, err := display.ConfirmExecution()
	if err != io.EOF {
		t.Errorf("Expected EOF error, got %v", err)
	}
}

func TestShowError(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	display := NewDisplay()
	testError := errors.New("test error message")
	display.ShowError(testError)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "Error: test error message\n"
	if output != expected {
		t.Errorf("Expected error output '%s', got '%s'", expected, output)
	}
}

func TestShowSuccess(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	display := NewDisplay()
	display.ShowSuccess("Command output here\n")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "Command output here\n"
	if output != expected {
		t.Errorf("Expected success output '%s', got '%s'", expected, output)
	}
}

func TestShowDangerWarning_Yes(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reader := bufio.NewReader(strings.NewReader("yes\n"))
	display := &Display{reader: reader}

	confirmed, err := display.ShowDangerWarning("Command contains 'rm -rf'")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("ShowDangerWarning() failed: %v", err)
	}

	if !confirmed {
		t.Error("Expected confirmation with 'yes' input")
	}

	if !strings.Contains(output, "WARNING") {
		t.Error("Expected warning message in output")
	}

	if !strings.Contains(output, "Command contains 'rm -rf'") {
		t.Error("Expected reason in output")
	}
}

func TestShowDangerWarning_No(t *testing.T) {
	testCases := []string{
		"no\n",
		"n\n",
		"y\n",     // Only full "yes" should confirm
		"\n",      // Empty should not confirm
		"maybe\n",
	}

	for _, input := range testCases {
		reader := bufio.NewReader(strings.NewReader(input))
		display := &Display{reader: reader}

		confirmed, err := display.ShowDangerWarning("Test reason")

		if err != nil && err != io.EOF {
			t.Errorf("ShowDangerWarning() with input '%s' failed: %v", strings.TrimSpace(input), err)
		}

		if confirmed {
			t.Errorf("Expected no confirmation with input '%s'", strings.TrimSpace(input))
		}
	}
}

func TestShowDangerWarning_FullYesRequired(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\n"))
	display := &Display{reader: reader}

	confirmed, _ := display.ShowDangerWarning("Test")

	if confirmed {
		t.Error("'y' alone should not confirm dangerous command, only 'yes' should")
	}
}
