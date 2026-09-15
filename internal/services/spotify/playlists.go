package spotify

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	playlist4pb "github.com/devgianlu/go-librespot/proto/spotify/playlist4"
	"github.com/devgianlu/go-librespot/session"
	"google.golang.org/protobuf/proto"
)

type Playlist struct {
	URI        string
	Name       string
	Owner      string
	CoverURL   string
	UpdatedAt  int64 // epoch ms; 0 if not available
	CreatedAt  int64 // epoch ms; 0 if not available
	TrackCount int32 // -1 if not available
}

func coverFromImageURL(s string) string {
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		return s
	}
	if !strings.HasPrefix(s, URIMosaicPrefix) {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(s, URIMosaicPrefix), ":")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return "https://i.scdn.co/image/" + parts[0]
}

func GetPlaylists(ctx context.Context, sess *session.Session, username string, usePublic bool) ([]Playlist, error) {
	isOwn := strings.EqualFold(username, sess.Username())
	if !usePublic && isOwn {
		pls := getRootlist(ctx, sess, username)
		if len(pls) > 0 {
			public, err := getPublicPlaylists(ctx, sess, username)
			if err == nil && len(public) > 0 {
				byURI := make(map[string]Playlist, len(public))
				for _, p := range public {
					byURI[p.URI] = p
				}
				for i := range pls {
					if named, ok := byURI[pls[i].URI]; ok {
						pls[i] = named
					}
				}
			}
			return enrichPlaylistsWithTracks(ctx, sess, pls), nil
		}
	}
	return getPublicPlaylists(ctx, sess, username)
}

func enrichPlaylistsWithTracks(ctx context.Context, sess *session.Session, pls []Playlist) []Playlist {
	const maxConc = DefaultEnrichConcurrency
	sem := make(chan struct{}, maxConc)
	type result struct {
		name       string
		cover      string
		updatedAt  int64
		createdAt  int64
		trackCount int32
	}
	results := make([]result, len(pls))
	var wg sync.WaitGroup
	for i := range pls {
		needsMeta := pls[i].Name == "" || pls[i].CoverURL == ""
		if !needsMeta && pls[i].UpdatedAt > 0 && pls[i].TrackCount >= 0 {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			meta, err := getPlaylistMetadata(ctx, sess, pls[idx].URI)
			if err == nil {
				results[idx] = result{
					name:       meta.name,
					cover:      meta.cover,
					updatedAt:  meta.updatedAt,
					createdAt:  meta.createdAt,
					trackCount: meta.trackCount,
				}
				if meta.cover != "" && meta.name != "" {
					return
				}
			}
			tracks, err2 := GetPlaylistTracksPage(ctx, sess, pls[idx].URI, 0, 1)
			if err2 != nil || len(tracks) == 0 {
				return
			}
			t := tracks[0]
			if results[idx].name == "" {
				results[idx].name = t.Album
			}
			if results[idx].cover == "" {
				results[idx].cover = t.CoverURL
			}
		}(i)
	}
	wg.Wait()
	out := make([]Playlist, 0, len(pls))
	for i, p := range pls {
		if p.Name == "" && results[i].name != "" {
			p.Name = results[i].name
		}
		if p.CoverURL == "" && results[i].cover != "" {
			p.CoverURL = results[i].cover
		}
		if results[i].updatedAt > p.UpdatedAt {
			p.UpdatedAt = results[i].updatedAt
		}
		if results[i].createdAt > 0 && p.CreatedAt == 0 {
			p.CreatedAt = results[i].createdAt
		}
		if p.TrackCount < 0 && results[i].trackCount > 0 {
			p.TrackCount = results[i].trackCount
		}
		if p.Name != "" || p.CoverURL != "" {
			out = append(out, p)
		}
	}
	return out
}

type playlistMeta struct {
	name       string
	cover      string
	updatedAt  int64
	createdAt  int64
	trackCount int32
}

func getPlaylistMetadata(ctx context.Context, sess *session.Session, uri string) (playlistMeta, error) {
	if !strings.HasPrefix(uri, URIPlaylistPrefix) {
		return playlistMeta{}, &TypeMismatchError{Expected: "playlist", Got: "", URI: uri}
	}
	id := strings.TrimPrefix(uri, URIPlaylistPrefix)
	hmURI := "hm://playlist/v2/playlist/" + url.PathEscape(id)
	resp, err := sess.Spclient().RequestHm(ctx, "GET", hmURI, nil, nil)
	if err != nil {
		return playlistMeta{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return playlistMeta{}, err
	}
	if resp.StatusCode != 200 {
		return playlistMeta{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var pl playlist4pb.SelectedListContent
	if err := proto.Unmarshal(body, &pl); err != nil {
		return playlistMeta{}, err
	}
	m := playlistMeta{}
	if pl.Attributes != nil {
		m.name = pl.Attributes.GetName()
		if len(pl.Attributes.GetPicture()) > 0 {
			m.cover = "https://i.scdn.co/image/" + hex.EncodeToString(pl.Attributes.GetPicture())
		}
	}
	m.updatedAt = pl.GetTimestamp()
	m.createdAt = pl.GetCreatedAt()
	m.trackCount = pl.GetLength()
	if m.name == "" {
		return m, fmt.Errorf("sin nombre")
	}
	return m, nil
}

func getRootlist(ctx context.Context, sess *session.Session, username string) []Playlist {
	escaped := url.PathEscape(username)
	candidates := []string{
		"hm://playlist/v2/user/" + escaped + "/rootlist",
		"hm://playlist/user/" + escaped + "/rootlist",
	}

	for _, hmURI := range candidates {
		resp, err := sess.Spclient().RequestHm(ctx, "GET", hmURI, nil, nil)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var content playlist4pb.SelectedListContent
		if err := proto.Unmarshal(body, &content); err != nil || content.Contents == nil {
			continue
		}

		var out []Playlist
		for _, item := range content.Contents.Items {
			if item == nil {
				continue
			}
			uri := item.GetUri()
			if strings.HasPrefix(uri, URIUserPrefix) {
				if idx := strings.LastIndex(uri, ":playlist:"); idx > 0 {
					uri = URIPlaylistPrefix + uri[idx+len(":playlist:"):]
				}
			}
			if !strings.HasPrefix(uri, URIPlaylistPrefix) {
				continue
			}
			p := Playlist{URI: uri, TrackCount: -1}
			if item.Attributes != nil {
				p.Owner = item.Attributes.GetAddedBy()
			}
			out = append(out, p)
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func getPublicPlaylists(ctx context.Context, sess *session.Session, username string) ([]Playlist, error) {
	path := fmt.Sprintf("/user-profile-view/v3/profile/%s/playlists", url.PathEscape(username))

	var result struct {
		PublicPlaylists []struct {
			URI       string `json:"uri"`
			Name      string `json:"name"`
			ImageURL  string `json:"image_url"`
			OwnerName string `json:"owner_name"`
		} `json:"public_playlists"`
	}
	if err := getJSON(ctx, sess, path, &result); err != nil {
		return nil, err
	}

	playlists := make([]Playlist, 0, len(result.PublicPlaylists))
	for _, item := range result.PublicPlaylists {
		playlists = append(playlists, Playlist{
			URI:        item.URI,
			Name:       item.Name,
			Owner:      item.OwnerName,
			CoverURL:   coverFromImageURL(item.ImageURL),
			TrackCount: -1,
		})
	}
	return playlists, nil
}
