//go:build !linux

package lab

import (
	"context"
	"fmt"
)

func shieldedAvailable(Config) error {
	return fmt.Errorf("%w: shielded runtime requires Linux", ErrUnavailable)
}
func runShielded(context.Context, Config, func(Observation)) error {
	return fmt.Errorf("%w: shielded runtime requires Linux", ErrUnavailable)
}
