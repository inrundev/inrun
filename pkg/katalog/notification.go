// pkg/katalog/notification.go
//
// Notification accessors on *Katalog.
//
// Precedence (highest → lowest):
//   1. Katalog YAML (k.Notification)
//   2. ENV vars via NotificationConfig (k.konfig.Notification())
//   3. Hard defaults coded below
//
// KatalogNotification uses optional fields (e.g. Interval) so we can detect
// "not declared" vs "explicitly set". ENV-level NotificationConfig provides
// capability defaults (SMTP, Slack) and fallback intervals.
//
// Notifications fire only when:
//   - a condition transitions false → true
//   - AND the effective interval has elapsed since the last send
//
// This file mirrors pkg/katalog/security.go for future-proofing.

package katalog

import (
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// notificationEnvDefaults returns the NotificationConfig from konfig,
// or a zero-value reader when konfig is not wired (e.g. tests).
func (k *Katalog) notificationEnvDefaults() interface {
	EmailEnabled() bool
	SlackEnabled() bool
	DefaultInterval() orktypes.Duration
	SMTPHost() string
	SMTPPort() int
	SMTPUser() string
	SMTPPass() string
	SMTPFrom() string
	SlackWebhook() string
} {
	return &envNotificationReader{k: k}
}

// envNotificationReader adapts *konfig.NotificationConfig through a small
// interface so notification.go does not import konfig directly.
type envNotificationReader struct{ k *Katalog }

func (r *envNotificationReader) EmailEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Notification().Email.Enabled
}
func (r *envNotificationReader) SlackEnabled() bool {
	if r.k.konfig == nil {
		return false
	}
	return r.k.konfig.Notification().Slack.Enabled
}
func (r *envNotificationReader) DefaultInterval() orktypes.Duration {
	if r.k.konfig == nil {
		return orktypes.Duration{Duration: 15 * 60 * 1e9} // 15m fallback
	}
	return orktypes.Duration{Duration: r.k.konfig.Notification().DefaultInterval}
}
func (r *envNotificationReader) SMTPHost() string {
	if r.k.konfig == nil {
		return ""
	}
	return r.k.konfig.Notification().Email.SMTPHost
}
func (r *envNotificationReader) SMTPPort() int {
	if r.k.konfig == nil {
		return 0
	}
	return r.k.konfig.Notification().Email.SMTPPort
}
func (r *envNotificationReader) SMTPUser() string {
	if r.k.konfig == nil {
		return ""
	}
	return r.k.konfig.Notification().Email.SMTPUser
}
func (r *envNotificationReader) SMTPPass() string {
	if r.k.konfig == nil {
		return ""
	}
	return r.k.konfig.Notification().Email.SMTPPass
}
func (r *envNotificationReader) SMTPFrom() string {
	if r.k.konfig == nil {
		return ""
	}
	return r.k.konfig.Notification().Email.From
}
func (r *envNotificationReader) SlackWebhook() string {
	if r.k.konfig == nil {
		return ""
	}
	return r.k.konfig.Notification().Slack.Webhook
}

//
// ── Effective notification helpers ─────────────────────────────────────────────
//

// IsEmailNotificationEnabled reports whether email notifications are possible.
//
// Precedence:
//
//	YAML team.email present → require SMTP env capability
//	YAML absent             → no email notifications
func (k *Katalog) IsEmailNotificationEnabled() bool {
	env := k.notificationEnvDefaults()

	// No teams → no notifications
	if !k.HasTeams() {
		return false
	}

	// No SMTP capability → disabled
	if !env.EmailEnabled() {
		return false
	}

	// At least one team must have email targets
	for _, team := range k.Notification.Teams {
		if len(team.Email) > 0 {
			return true
		}
	}
	return false
}

// IsSlackNotificationEnabled reports whether Slack notifications are possible.
func (k *Katalog) IsSlackNotificationEnabled() bool {
	env := k.notificationEnvDefaults()

	if !k.HasTeams() {
		return false
	}
	if !env.SlackEnabled() {
		return false
	}
	for _, team := range k.Notification.Teams {
		if len(team.Slack) > 0 {
			return true
		}
	}
	return false
}

// NotificationInterval returns the effective interval for a team.
//
// Precedence:
//
//	YAML team.interval > YAML defaults.interval > ENV defaultInterval > hard default
func (k *Katalog) NotificationInterval(teamName string) orktypes.Duration {
	env := k.notificationEnvDefaults()

	// 1. Team override
	if k.Notification != nil {
		if team, ok := k.Notification.Teams[teamName]; ok {
			if team.Interval.Duration > 0 {
				return team.Interval
			}
		}

		// 2. YAML defaults
		if k.Notification.Defaults != nil && k.Notification.Defaults.Interval.Duration > 0 {
			return k.Notification.Defaults.Interval
		}
	}

	// 3. ENV default
	if env.DefaultInterval().Duration > 0 {
		return env.DefaultInterval()
	}

	// 4. Hard fallback: 15m
	return orktypes.Duration{Duration: 15 * 60 * 1e9}
}

// SMTPConfig returns the effective SMTP configuration (from ENV only).
func (k *Katalog) SMTPConfig() (host string, port int, user, pass, from string) {
	env := k.notificationEnvDefaults()
	return env.SMTPHost(), env.SMTPPort(), env.SMTPUser(), env.SMTPPass(), env.SMTPFrom()
}

// SlackWebhook returns the effective Slack webhook URL (from ENV only).
func (k *Katalog) SlackWebhook() string {
	return k.notificationEnvDefaults().SlackWebhook()
}

// HasNotification returns whether a katalog has notification configured or not
func (k *Katalog) HasNotification() bool {
	return k.Notification != nil
}

// IsNotificationStandalone reports whether notification dispatch runs directly
// on the runtime without a gateway.
//
// Precedence:
//
//	YAML notification.standalone declared → use YAML value
//	Not in a Kubernetes cluster (local dev / CLI) → implicit true
//	In-cluster, no explicit declaration → false (gateway required)
func (k *Katalog) IsNotificationStandalone() bool {
	if k.Notification != nil && k.Notification.Standalone != nil {
		return *k.Notification.Standalone
	}
	// Outside a cluster there is no gateway infrastructure.
	// Fall back to standalone so local dev and CLI usage work without one.
	return !utils.IsRunningInPod()
}

// HasTeams returns whether a katalog has teams configured or not
func (k *Katalog) HasTeams() bool {
	return k.Notification != nil && len(k.Notification.Teams) > 0
}
