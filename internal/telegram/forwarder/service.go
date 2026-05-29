package forwarder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
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
	s.log.Info("received zenfl message", messageLogFields(m)...)
	s.logRawMessageJSON(m)
	outgoingText := buildOutgoingTextWithJobLink(m)

	for _, t := range s.targets {
		randomID := rand.New(rand.NewSource(time.Now().UnixNano())).Int63()
		_, err := s.api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
			Peer:        t.Peer,
			Message:     outgoingText,
			Entities:    m.Entities,
			ReplyMarkup: nil,
			RandomID:    randomID,
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

func messageLogFields(m *tg.Message) []zap.Field {
	fields := []zap.Field{
		zap.Int("id", m.ID),
		zap.Int("date_unix", m.Date),
		zap.String("from", peerToString(m.FromID)),
		zap.String("peer", peerToString(m.PeerID)),
		zap.Bool("out", m.Out),
		zap.Bool("silent", m.Silent),
		zap.Bool("mentioned", m.Mentioned),
		zap.Bool("media_unread", m.MediaUnread),
		zap.Bool("post", m.Post),
		zap.Bool("legacy", m.Legacy),
		zap.Int("edit_date_unix", m.EditDate),
		zap.Int("forwards", m.Forwards),
		zap.Int("views", m.Views),
		zap.Int("replies", repliesCount(m.Replies)),
		zap.Bool("has_text", m.Message != ""),
		zap.String("text", m.Message),
		zap.String("media_type", typeName(m.Media)),
		zap.String("reply_to_type", typeName(m.ReplyTo)),
		zap.String("fwd_from_type", typeName(m.FwdFrom)),
		zap.String("reply_markup_type", typeName(m.ReplyMarkup)),
		zap.Int("entities_count", len(m.Entities)),
		zap.Any("raw_message", m),
	}

	return fields
}

func peerToString(peer tg.PeerClass) string {
	if peer == nil {
		return "<nil>"
	}
	switch p := peer.(type) {
	case *tg.PeerUser:
		return fmt.Sprintf("user:%d", p.UserID)
	case *tg.PeerChat:
		return fmt.Sprintf("chat:%d", p.ChatID)
	case *tg.PeerChannel:
		return fmt.Sprintf("channel:%d", p.ChannelID)
	default:
		return reflect.TypeOf(peer).String()
	}
}

func (s *Service) logRawMessageJSON(m *tg.Message) {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		s.log.Warn("failed to marshal message json", zap.Int("message_id", m.ID), zap.Error(err))
		return
	}
	s.log.Info("received zenfl message json", zap.Int("message_id", m.ID), zap.String("json", string(raw)))
}

func typeName(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	return reflect.TypeOf(v).String()
}

func repliesCount(r tg.MessageReplies) int {
	return r.Replies
}

func buildOutgoingTextWithJobLink(m *tg.Message) string {
	base := strings.TrimRight(m.Message, "\n \t")
	link := extractFirstURLButtonLink(m.ReplyMarkup)
	if link == "" {
		return base
	}
	if strings.Contains(base, link) {
		return base
	}
	return base + "\n\nUpwork job link: " + link
}

func extractFirstURLButtonLink(markup tg.ReplyMarkupClass) string {
	inline, ok := markup.(*tg.ReplyInlineMarkup)
	if !ok {
		return ""
	}
	for _, row := range inline.Rows {
		for _, btn := range row.Buttons {
			if u, ok := btn.(*tg.KeyboardButtonURL); ok {
				return strings.TrimSpace(u.URL)
			}
		}
	}
	return ""
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
