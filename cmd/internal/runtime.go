package internal

import (
	"context"

	"github.com/inrundev/inrun/pkg/config"
	"github.com/inrundev/inrun/pkg/labels"
	"github.com/inrundev/inrun/pkg/logger"
	"github.com/inrundev/inrun/pkg/merger"
	"github.com/inrundev/inrun/pkg/runtime/leader"
	"github.com/inrundev/inrun/pkg/template"
	"github.com/inrundev/inrun/pkg/utils"
	"github.com/inrundev/inrun/pkg/version"
)

func RunRuntime(kfg *config.Config, m *merger.Merger, ctx context.Context) {
	// Runtime facts, available to every template expression as .inrun.*. Set on
	// the context that construction, startup and the coordinator all share.
	ctx = template.ContextWithInrunContext(ctx, template.NewInrunContext(
		kfg.Cluster().Namespace(),
		version.Version,
	))

	// create domain component and build inrun
	startup := constructRuntime(kfg, m, ctx)

	// ── Start ─────────────────────────────────────────────────────────────────
	go func() {
		if err := startup.inrun.Start(ctx); err != nil {
			logger.Fatal().AnErr("inrun startup error", err)
			utils.Exit(err)
		}
	}()

	ko := leader.NewLeaderElection(
		startup.kube,
		startup.event,
		func(ctx context.Context) { startup.kord.Coordinate(ctx) },
		func(leader string) {
			// Banner prints here — leader is the actual winner
			printBanner(startup, leader)
		},
		leader.Options{
			Namespace:     kfg.Leader().Namespace(),
			LeaseDuration: kfg.Leader().LeaseDuration(),
			RenewDeadline: kfg.Leader().RenewDeadline(),
			RetryPeriod:   kfg.Leader().RetryPeriod(),
			Labels:        labels.WithDeletionProtection(nil),
		})

	// start leader election as postStartHook after  inrun is ready
	startup.inrun.AddPostStartHook(ko, func(ctx context.Context) {
		logger.Info().Msg("starting leader election...")
		ko.Start(ctx)
	})

	// Keep running until cancelled
	startup.inrun.Wait()
}
