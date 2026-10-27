// Package contracts is the registry of the modules' contracts: the small interfaces through which
// one module calls another (plain Go calls, no network). Each module sets its own contract during
// RegisterContracts; the others hold the *Contracts and call it from then on.
package contracts

import (
	"errors"

	catalog "github.com/erdira/homegym/backend/modules/catalog/api/module/client"
	identity "github.com/erdira/homegym/backend/modules/identity/api/module/client"
	planning "github.com/erdira/homegym/backend/modules/planning/api/module/client"
	tracking "github.com/erdira/homegym/backend/modules/tracking/api/module/client"
)

// Contracts holds every module's contract.
type Contracts struct {
	Identity identity.Identity
	Catalog  catalog.Catalog
	Planning planning.Planning
	Tracking tracking.Tracking
}

// Verify reports the contracts no module has set.
func (c *Contracts) Verify() error {
	var err error
	if c.Identity == nil {
		err = errors.Join(err, errors.New("identity contract is not set"))
	}
	if c.Catalog == nil {
		err = errors.Join(err, errors.New("catalog contract is not set"))
	}
	if c.Planning == nil {
		err = errors.Join(err, errors.New("planning contract is not set"))
	}
	if c.Tracking == nil {
		err = errors.Join(err, errors.New("tracking contract is not set"))
	}
	return err
}
