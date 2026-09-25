package youtube

import (
	"encoding/json"
	"strconv"
	"strings"
)

type format struct {
	Itag     int    `json:"itag"`
	MimeType string `json:"mimeType"`
	URL      string `json:"url"`
}

type playerResponse struct {
	VideoDetails struct {
		Title         string      `json:"title"`
		LengthSeconds json.Number `json:"lengthSeconds"`
		Author        string      `json:"author"`
		ChannelID     string      `json:"channelId"`
		Thumbnail     thumbnail   `json:"thumbnail"`
	} `json:"videoDetails"`
	StreamingData struct {
		Formats         []format `json:"formats"`
		AdaptiveFormats []format `json:"adaptiveFormats"`
	} `json:"streamingData"`
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"playabilityStatus"`
}

type compactVideoRenderer struct {
	VideoID    string    `json:"videoId"`
	Title      text      `json:"title"`
	ByLine     text      `json:"shortBylineText"`
	LengthText text      `json:"lengthText"`
	Thumbnail  thumbnail `json:"thumbnail"`
}

func (v compactVideoRenderer) duration() int {
	parts := strings.Split(v.LengthText.first(), ":")
	total := 0
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0
		}
		total = total*60 + value
	}
	return total
}

type searchResponse struct {
	Contents struct {
		SectionListRenderer struct {
			Contents []struct {
				ItemSectionRenderer struct {
					Contents []struct {
						CompactVideoRenderer compactVideoRenderer `json:"compactVideoRenderer"`
					} `json:"contents"`
				} `json:"itemSectionRenderer"`
			} `json:"contents"`
		} `json:"sectionListRenderer"`
	} `json:"contents"`
}

type expireAndDuration struct {
	ExpireAt int64
	Duration int
}

type text struct {
	Runs []struct {
		Text               string `json:"text"`
		NavigationEndpoint struct {
			BrowseEndpoint struct {
				BrowseID string `json:"browseId"`
			} `json:"browseEndpoint"`
		} `json:"navigationEndpoint"`
	} `json:"runs"`
}

func (t text) first() string {
	if len(t.Runs) == 0 {
		return ""
	}
	return t.Runs[0].Text
}

// browseID returns the channel id carried by the first run's navigation
// endpoint, when present (byline and owner text link to their channel).
func (t text) browseID() string {
	if len(t.Runs) == 0 {
		return ""
	}
	return t.Runs[0].NavigationEndpoint.BrowseEndpoint.BrowseID
}

type thumbnail struct {
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
}

func (t thumbnail) url() string {
	if len(t.Thumbnails) == 0 {
		return ""
	}
	return capThumbnail(t.Thumbnails[len(t.Thumbnails)-1].URL)
}

// capThumbnail caps i.ytimg.com thumbnails at hqdefault (480px). Art is shown
// at 40-220px, but the API hands out maxresdefault/hq720 (~1280px), and each
// one decodes to a few MB in WebKit. Smaller variants are left untouched so
// nothing is ever upscaled.
func capThumbnail(url string) string {
	if !strings.HasPrefix(url, "https://i.ytimg.com/") {
		return url
	}
	slash := strings.LastIndexByte(url, '/')
	dot := strings.LastIndexByte(url, '.')
	if slash < 0 || dot <= slash {
		return url
	}
	switch url[slash+1 : dot] {
	case "hq720", "sddefault", "maxresdefault", "maxres":
		return url[:slash+1] + "hqdefault" + url[dot:]
	}
	return url
}

type playlistHeaderRenderer struct {
	Title     text `json:"title"`
	OwnerText text `json:"ownerText"`
	Banner    struct {
		HeroPlaylistThumbnailRenderer struct {
			Thumbnail thumbnail `json:"thumbnail"`
		} `json:"heroPlaylistThumbnailRenderer"`
	} `json:"playlistHeaderBanner"`
}

type playlistVideoRenderer struct {
	VideoId       string      `json:"videoId"`
	Title         text        `json:"title"`
	ByLine        text        `json:"shortBylineText"`
	LengthSeconds json.Number `json:"lengthSeconds"`
	Thumbnail     thumbnail   `json:"thumbnail"`
	IsPlayable    *bool       `json:"isPlayable"`
}

type playlistVideoListItem struct {
	PlaylistVideoRenderer playlistVideoRenderer `json:"playlistVideoRenderer"`
}

type playlistVideoListRenderer struct {
	Contents      []playlistVideoListItem    `json:"contents"`
	Continuations []playlistNextContinuation `json:"continuations"`
}

type playlistNextContinuation struct {
	NextContinuationData struct {
		Continuation string `json:"continuation"`
	} `json:"nextContinuationData"`
}

func (l playlistVideoListRenderer) next() string {
	if len(l.Continuations) == 0 {
		return ""
	}
	return l.Continuations[0].NextContinuationData.Continuation
}

type playlistBrowseResponse struct {
	Header struct {
		PlaylistHeaderRenderer playlistHeaderRenderer `json:"playlistHeaderRenderer"`
	} `json:"header"`
	Contents struct {
		SingleColumnBrowseResultsRenderer struct {
			Tabs []struct {
				TabRenderer struct {
					Content struct {
						SectionListRenderer struct {
							Contents []struct {
								PlaylistVideoListRenderer playlistVideoListRenderer `json:"playlistVideoListRenderer"`
							} `json:"contents"`
						} `json:"sectionListRenderer"`
					} `json:"content"`
				} `json:"tabRenderer"`
			} `json:"tabs"`
		} `json:"singleColumnBrowseResultsRenderer"`
	} `json:"contents"`
}

type playlistContinuationResponse struct {
	ContinuationContents struct {
		PlaylistVideoListContinuation playlistVideoListRenderer `json:"playlistVideoListContinuation"`
	} `json:"continuationContents"`
}

type compactPlaylistRenderer struct {
	PlaylistId string    `json:"playlistId"`
	Title      text      `json:"title"`
	ByLine     text      `json:"shortBylineText"`
	Thumbnail  thumbnail `json:"thumbnail"`
}

type playlistSearchResponse struct {
	Contents struct {
		SectionListRenderer struct {
			Contents []struct {
				ItemSectionRenderer struct {
					Contents []struct {
						CompactPlaylistRenderer compactPlaylistRenderer `json:"compactPlaylistRenderer"`
					} `json:"contents"`
				} `json:"itemSectionRenderer"`
			} `json:"contents"`
		} `json:"sectionListRenderer"`
	} `json:"contents"`
}

// itemSection holds a run of compact playlist or video renderers plus its
// continuation, shared by channel tabs and their continuation responses.
type itemSection struct {
	Contents []struct {
		CompactPlaylistRenderer compactPlaylistRenderer `json:"compactPlaylistRenderer"`
		CompactVideoRenderer    compactVideoRenderer    `json:"compactVideoRenderer"`
	} `json:"contents"`
	Continuations []playlistNextContinuation `json:"continuations"`
}

type channelBrowseResponse struct {
	Header struct {
		C4TabbedHeaderRenderer struct {
			Title  string    `json:"title"`
			Avatar thumbnail `json:"avatar"`
		} `json:"c4TabbedHeaderRenderer"`
	} `json:"header"`
	Contents struct {
		SingleColumnBrowseResultsRenderer struct {
			Tabs []struct {
				TabRenderer struct {
					Content struct {
						SectionListRenderer struct {
							Contents []struct {
								ItemSectionRenderer itemSection `json:"itemSectionRenderer"`
							} `json:"contents"`
						} `json:"sectionListRenderer"`
					} `json:"content"`
				} `json:"tabRenderer"`
			} `json:"tabs"`
		} `json:"singleColumnBrowseResultsRenderer"`
	} `json:"contents"`
}

type channelContinuationResponse struct {
	ContinuationContents struct {
		ItemSectionContinuation itemSection `json:"itemSectionContinuation"`
	} `json:"continuationContents"`
}

type resolveURLResponse struct {
	Endpoint struct {
		BrowseEndpoint struct {
			BrowseID string `json:"browseId"`
		} `json:"browseEndpoint"`
	} `json:"endpoint"`
}
