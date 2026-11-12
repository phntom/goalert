package sources

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

// terminalAuth implements auth.UserAuthenticator prompting the terminal for
// input.
type terminalAuth struct {
	PhoneNumber string // optional, will be prompted if empty
}

var _ auth.UserAuthenticator = terminalAuth{}

func (terminalAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("signing up not implemented in Terminal")
}

func (terminalAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (terminalAuth) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a terminalAuth) Phone(_ context.Context) (string, error) {
	if a.PhoneNumber != "" {
		return a.PhoneNumber, nil
	}
	fmt.Print("Enter phone in international format (e.g. +1234567890): ")
	phone, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(phone), nil
}

func (terminalAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	bytePwd, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePwd)), nil
}
