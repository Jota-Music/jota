package follows

// Users is the set of usernames an account follows. Follows are local to Jota
// and scoped by the owning account, so different accounts keep separate lists.
type Users []string
