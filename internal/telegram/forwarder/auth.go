package forwarder

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func (s *Service) authFlow() auth.Flow {
	reader := bufio.NewReader(os.Stdin)
	return auth.NewFlow(
		auth.Constant(s.cfg.Phone,
			auth.CodeAuthenticatorFunc(func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
				fmt.Print("Enter Telegram code: ")
				code, err := reader.ReadString('\n')
				if err != nil {
					return "", err
				}
				return strings.TrimSpace(code), nil
			}),
			auth.PasswordAuthenticatorFunc(func(ctx context.Context) (string, error) {
				fmt.Print("Enter 2FA password: ")
				pw, err := reader.ReadString('\n')
				if err != nil {
					return "", err
				}
				return strings.TrimSpace(pw), nil
			}),
		),
		auth.SendCodeOptions{},
	)
}
