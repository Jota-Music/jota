package app

// GetOrder returns the user's custom order of ids for a shelf, by scope and,
// for a shelf that belongs to one, by account.
func (a *App) GetOrder(scope string, account string) ([]string, error) {
	return a.Order.List(scope, account)
}

// SaveOrder stores the user's custom order of ids for a shelf.
func (a *App) SaveOrder(scope string, account string, ids []string) error {
	return a.Order.Save(scope, account, ids)
}
