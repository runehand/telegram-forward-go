package forwarder

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/tg"
)

func (s *Service) resolvePeers(ctx context.Context) error {
	zenflPeer, zenflID, err := s.resolveUsernameToPeer(ctx, s.cfg.ZenflUsername)
	if err != nil {
		return fmt.Errorf("failed to resolve ZENFL_USERNAME: %w", err)
	}
	s.zenflID = zenflID
	s.zenflPeer = zenflPeer

	targets := make([]target, 0, len(s.cfg.TargetUsernames))
	for _, username := range s.cfg.TargetUsernames {
		peer, _, err := s.resolveUsernameToPeer(ctx, username)
		if err != nil {
			return fmt.Errorf("failed to resolve target %q: %w", username, err)
		}
		targets = append(targets, target{Name: username, Peer: peer})
	}
	s.targets = targets

	return nil
}

func (s *Service) resolveUsernameToPeer(ctx context.Context, username string) (tg.InputPeerClass, int64, error) {
	resolved, err := s.api.ContactsResolveUsername(ctx, username)
	if err != nil {
		return nil, 0, err
	}

	switch p := resolved.Peer.(type) {
	case *tg.PeerUser:
		for _, u := range resolved.Users {
			if user, ok := u.(*tg.User); ok && user.ID == p.UserID {
				return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}, user.ID, nil
			}
		}
		return nil, 0, errors.New("resolved user missing from users list")
	case *tg.PeerChat:
		return &tg.InputPeerChat{ChatID: p.ChatID}, p.ChatID, nil
	case *tg.PeerChannel:
		for _, c := range resolved.Chats {
			if ch, ok := c.(*tg.Channel); ok && ch.ID == p.ChannelID {
				return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, ch.ID, nil
			}
		}
		return nil, 0, errors.New("resolved channel missing from chats list")
	default:
		return nil, 0, fmt.Errorf("unsupported peer type %T", p)
	}
}
