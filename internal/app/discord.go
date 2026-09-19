package app

import (
	"encoding/json"
	"log"
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/services/discord"
)

var settings = kv.UseBucket("settings")

// Media mirrors the frontend media payload from media-session.ts.
type Media struct {
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	Artwork  string  `json:"artwork"`
	Playing  bool    `json:"playing"`
	Duration float64 `json:"duration"`
	Position float64 `json:"position"`
}

// discordState tracks the last SET_ACTIVITY so we only re-send when the bar
// would meaningfully change (track, play state, duration becoming known, seek).
type discordState struct {
	fingerprint string
	position    float64
	setAt       time.Time
	timed       bool
}

func (a *App) SetDiscordEnabled(enabled bool) error {
	if runtime.GOOS == "android" {
		return nil
	}
	if !enabled {
		a.Discord.Clear()
		a.discordMu.Lock()
		a.discord = nil
		a.discordMu.Unlock()
	}
	return settings.SetObject("discord", enabled)
}

func (a *App) DiscordEnabled() bool {
	if runtime.GOOS == "android" {
		return false
	}
	var enabled bool
	if err := settings.GetObject("discord", &enabled); err != nil {
		return false
	}
	return enabled
}

func (a *App) UpdateDiscordPresence(payload string) error {
	if !a.DiscordEnabled() {
		return nil
	}
	var m Media
	if err := json.Unmarshal([]byte(payload), &m); err != nil || m.Title == "" {
		return nil
	}
	fingerprint := m.Title + "\x00" + m.Artist + "\x00" + m.Album + "\x00" +
		m.Artwork + "\x00" + strconv.FormatBool(m.Playing)

	a.discordMu.Lock()
	needsSet := a.needsDiscordSet(a.discord, fingerprint, m)
	a.discordMu.Unlock()
	if !needsSet {
		return nil
	}

	if err := a.Discord.Set(a.activity(m)); err != nil {
		msg := err.Error()
		a.discordMu.Lock()
		if msg != a.discordErr {
			a.discordErr = msg
			a.discordMu.Unlock()
			log.Printf("discord: presence update failed: %v", err)
		} else {
			a.discordMu.Unlock()
		}
	} else {
		a.discordMu.Lock()
		a.discord = &discordState{
			fingerprint: fingerprint,
			position:    m.Position,
			setAt:       time.Now(),
			timed:       m.Playing && m.Duration > 0,
		}
		a.discordErr = ""
		a.discordMu.Unlock()
	}
	return nil
}

func (a *App) needsDiscordSet(st *discordState, fingerprint string, m Media) bool {
	if st == nil || st.fingerprint != fingerprint {
		return true
	}
	if !m.Playing {
		return false
	}
	if m.Duration > 0 && !st.timed {
		return true
	}
	if !st.timed {
		return false
	}
	expected := st.position + time.Since(st.setAt).Seconds()
	return math.Abs(m.Position-expected) > 20
}

func (a *App) ClearDiscordPresence() error {
	a.Discord.Clear()
	a.discordMu.Lock()
	a.discord = nil
	a.discordMu.Unlock()
	return nil
}

func (a *App) activity(m Media) discord.Activity {
	activity := discord.Activity{
		Type:    2,
		Details: m.Title,
		State:   m.Artist,
		Assets:  &discord.Assets{LargeText: m.Album},
	}
	if m.Artwork != "" {
		activity.Assets.LargeImage = m.Artwork
	}
	if m.Playing && m.Duration > 0 && m.Position >= 0 && m.Position < m.Duration {
		now := time.Now().UnixMilli()
		activity.Timestamps = &discord.Timestamps{
			Start: now - int64(m.Position*1000),
			End:   now + int64((m.Duration-m.Position)*1000),
		}
	}
	return activity
}
