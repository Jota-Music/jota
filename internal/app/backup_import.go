package app

import "github.com/Jota-Music/jota/internal/backup"

// ImportBackup restores every section a backup file carries, letting the file
// win over local deletions because the user asked for it.
func (a *App) ImportBackup(data string) (backup.Stats, error) {
	stats, err := a.Backup.Restore(data)
	if err != nil {
		return backup.Stats{}, err
	}
	a.syncInBackground()
	return stats, nil
}
