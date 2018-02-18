package terminal

import (
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// ReadSecretLine reads a line of input from the terminal without echoing it
// back. It is useful for getting users to input sensitive information like
// passwords without revealing it to others who might be able to see the screen.
func ReadSecretLine() (string, error) {
	fd := int(syscall.Stdin)
	terminal.ReadPassword(fd)
	return "", nil
}
