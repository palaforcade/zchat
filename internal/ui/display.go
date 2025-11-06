package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Display struct {
	reader *bufio.Reader
}

// NewDisplay creates a new display with stdin reader
func NewDisplay() *Display {
	return &Display{
		reader: bufio.NewReader(os.Stdin),
	}
}

// ShowCommand prints the generated command
func (d *Display) ShowCommand(command string) {
	fmt.Printf("Command: %s\n", command)
}

// ConfirmExecution prompts the user to confirm execution
func (d *Display) ConfirmExecution() (bool, error) {
	fmt.Print("Execute? [Y/n]: ")

	input, err := d.reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Accept y, Y, yes, or empty (just pressing enter)
	if input == "" || input == "y" || input == "yes" {
		return true, nil
	}

	return false, nil
}

// ShowError prints an error message to stderr
func (d *Display) ShowError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// ShowSuccess prints command output
func (d *Display) ShowSuccess(output string) {
	fmt.Print(output)
}

// ShowDangerWarning shows a warning about dangerous commands and asks for explicit confirmation
func (d *Display) ShowDangerWarning(reason string) (bool, error) {
	fmt.Println("\n⚠️  WARNING: Dangerous command detected!")
	fmt.Printf("Reason: %s\n", reason)
	fmt.Print("Are you SURE you want to execute this? [yes/no]: ")

	input, err := d.reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Require full "yes" to proceed
	return input == "yes", nil
}
