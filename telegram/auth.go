package telegram

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

type terminalAuth struct {
	reader *bufio.Reader
}

func (a *terminalAuth) Phone(_ context.Context) (string, error) {
	fmt.Print("Enter phone number (with country code): ")
	phone, err := a.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(phone), nil
}

func (a *terminalAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	pass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // newline after hidden input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(pass)), nil
}

func (a *terminalAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter auth code: ")
	code, err := a.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a *terminalAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	fmt.Println("Terms of Service accepted.")
	return nil
}

func (a *terminalAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("sign up not supported, use an existing account")
}

// NewAuthFlow returns an auth flow that prompts in the terminal.
func NewAuthFlow() auth.Flow {
	return auth.NewFlow(
		&terminalAuth{reader: bufio.NewReader(os.Stdin)},
		auth.SendCodeOptions{},
	)
}
