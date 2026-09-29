package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// NotificationState tracks per condition+team last-send timestamps for throttling.
type NotificationState struct {
	// key: operatorName + "|" + conditionKey + "|" + teamName
	LastSent map[string]time.Time
}

// NotificationStack combines Katalog context, throttle state, and the active
// Notifier. One instance per generic.Reconciler when notification is enabled.
type NotificationStack struct {
	Katalog  *katalog.Katalog
	State    *NotificationState
	Notifier Notifier
}

func NewNotificationState() *NotificationState {
	return &NotificationState{
		LastSent: make(map[string]time.Time),
	}
}

// NewNotificationStack creates a ready-to-use stack.
// notifier is either a DirectNotifier (standalone) or GatewayNotifier (gateway path).
func NewNotificationStack(kat *katalog.Katalog, notifier Notifier) *NotificationStack {
	return &NotificationStack{
		Katalog:  kat,
		State:    NewNotificationState(),
		Notifier: notifier,
	}
}

// conditionKey is a stable identifier for a condition within an operatorBox.
func conditionKey(cond orktypes.Condition) string {
	op, val := orktypes.ResolveConditionOp(cond)
	return fmt.Sprintf("%s|%s|%s", cond.Field, op, val)
}

// ProcessConditionNotifications enforces per-team throttle and fires an Event
// via s.Notifier for every team whose interval has elapsed.
//
// Call this after a condition has already been evaluated as true.
func (s *NotificationStack) ProcessConditionNotifications(
	ctx context.Context,
	data map[string]interface{},
	cond orktypes.Condition,
	now time.Time,
) {
	if cond.Notify == nil || len(cond.Notify.Teams) == 0 {
		return
	}
	k := s.Katalog
	if !k.HasTeams() {
		return
	}

	ck := conditionKey(cond)

	for _, teamName := range cond.Notify.Teams {
		team, ok := k.Notification.Teams[teamName]
		if !ok || team == nil {
			continue
		}

		key := fmt.Sprintf("%s|%s|%s", k.Meta().Name, ck, teamName)
		last := s.State.LastSent[key]
		interval := k.NotificationInterval(teamName)

		if !last.IsZero() && now.Sub(last) < interval.Duration {
			continue
		}

		message := cond.Notify.Message
		if message == "" {
			message = k.Notification.EffectiveMessage(teamName)
		}

		ev := Event{
			KatalogName: k.Meta().Name,
			CondKey:     ck,
			TeamName:    teamName,
			Subject:     fmt.Sprintf("%s condition triggered", cond.Field),
			Message:     message,
			Timestamp:   now,
			Data:        data,
		}

		if err := s.Notifier.Dispatch(ctx, ev); err != nil {
			continue
		}
		s.State.LastSent[key] = now
	}
}

// dispatchTeam fans out to email/slack for a single team.
// Called by DirectNotifier (standalone path) and the gateway /notify handler.
func dispatchTeam(
	ctx context.Context,
	k *katalog.Katalog,
	teamName string,
	team *orktypes.NotificationTeam,
	subject, message string,
	data map[string]interface{},
) {

	// Emails
	if len(team.Email) > 0 && k.IsEmailNotificationEnabled() {
		host, port, user, pass, from := k.SMTPConfig()
		if host != "" && user != "" && pass != "" {
			cfg := SMTPConfig{
				Host: host,
				Port: fmt.Sprintf("%d", port),
				User: user,
				Pass: pass,
				From: from,
			}
			_ = sendEmailNotification(ctx, cfg, team.Email, k.Meta().Name, teamName, subject, message)
		}
	}

	// Slack
	if len(team.Slack) > 0 && k.IsSlackNotificationEnabled() {
		webhook := k.Notification.EffectiveSlackWebhook(teamName)
		if webhook == "" {
			webhook = k.SlackWebhook()
		}
		if webhook != "" {
			_ = sendSlackNotification(ctx, webhook, team.Slack, k.Meta().Name, teamName, message, "warning")
		}
	}
}
