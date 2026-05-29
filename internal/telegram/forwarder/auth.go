package forwarder

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func (s *Service) authFlow() auth.Flow {
	return auth.NewFlow(&interactiveAuth{
		phone:  s.cfg.Phone,
		reader: bufio.NewReader(os.Stdin),
	}, auth.SendCodeOptions{})
}

type interactiveAuth struct {
	phone  string
	reader *bufio.Reader
}

func (a *interactiveAuth) Phone(context.Context) (string, error) {
	return a.phone, nil
}

func (a *interactiveAuth) Code(context.Context, *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter Telegram code: ")
	code, err := a.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a *interactiveAuth) Password(context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	pw, err := a.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pw), nil
}

func (a *interactiveAuth) AcceptTermsOfService(context.Context, tg.HelpTermsOfService) error {
	return errors.New("sign-up flow is not supported")
}

func (a *interactiveAuth) SignUp(context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("sign-up flow is not supported")
}
