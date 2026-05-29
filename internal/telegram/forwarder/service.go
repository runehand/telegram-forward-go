package forwarder

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"go.uber.org/zap"

	"zenfl-forwarder/internal/config"
)

type Service struct {
	cfg       config.TelegramConfig
	log       *zap.Logger
	targets   []target
	zenflID   int64
	zenflPeer tg.InputPeerClass
	api       *tg.Client
}

func NewService(cfg config.TelegramConfig, logger *zap.Logger) *Service {
	return &Service{cfg: cfg, log: logger}
}

func (s *Service) Run(ctx context.Context) error {
	storage := &telegram.FileSessionStorage{Path: s.cfg.SessionFile}

	updates := tg.NewUpdateDispatcher()
	updates.OnNewMessage(func(ctx context.Context, _ tg.Entities, update *tg.UpdateNewMessage) error {
		return s.handleMessage(ctx, update.Message)
	})
	updates.OnNewChannelMessage(func(ctx context.Context, _ tg.Entities, update *tg.UpdateNewChannelMessage) error {
		return s.handleMessage(ctx, update.Message)
	})

	client := telegram.NewClient(s.cfg.APIID, s.cfg.APIHash, telegram.Options{
		SessionStorage: storage,
		Logger:         s.log,
		UpdateHandler:  updates,
	})

	err := client.Run(ctx, func(ctx context.Context) error {
		s.api = client.API()

		if err := client.Auth().IfNecessary(ctx, s.authFlow()); err != nil {
			return err
		}
		if err := s.resolvePeers(ctx); err != nil {
			return err
		}

		s.log.Info("forwarder is running", zap.String("zenfl", s.cfg.ZenflUsername), zap.Int("targets", len(s.targets)))
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (s *Service) handleMessage(ctx context.Context, msg tg.MessageClass) error {
	m, ok := msg.(*tg.Message)
	if !ok || m.Out {
		return nil
	}

	if extractSenderID(m) != s.zenflID {
		return nil
	}

	for _, t := range s.targets {
		randomID := rand.New(rand.NewSource(time.Now().UnixNano())).Int63()
		_, err := s.api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
			FromPeer: s.zenflPeer,
			ID:       []int{m.ID},
			RandomID: []int64{randomID},
			ToPeer:   t.Peer,
		})
		if err != nil {
			if tgerr.Is(err, "FLOOD_WAIT") {
				s.log.Warn("flood wait while forwarding", zap.String("target", t.Name), zap.Error(err))
				continue
			}
			s.log.Error("forward failed", zap.String("target", t.Name), zap.Int("message_id", m.ID), zap.Error(err))
			continue
		}
		s.log.Info("forwarded", zap.Int("message_id", m.ID), zap.String("target", t.Name))
	}

	return nil
}

func extractSenderID(m *tg.Message) int64 {
	if m.FromID != nil {
		switch from := m.FromID.(type) {
		case *tg.PeerUser:
			return from.UserID
		case *tg.PeerChannel:
			return from.ChannelID
		case *tg.PeerChat:
			return from.ChatID
		}
	}

	switch p := m.PeerID.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChannel:
		return p.ChannelID
	case *tg.PeerChat:
		return p.ChatID
	default:
		return 0
	}
}
