package ordering

// Order is the user's custom sequence of playlist ids for the library shelf. It
// is local to Jota and scoped by the owning account, so different accounts keep
// separate orders; it only holds ids and never edits the source playlists.
type Order []string
