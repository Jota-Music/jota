package ordering

// Order is the user's custom sequence of ids for one shelf, such as the library
// or the following shelf. It is local to Jota and keyed by the shelf's scope, so
// every shelf keeps its own order; it only holds ids and never edits the source
// collection.
type Order []string
