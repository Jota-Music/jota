package youtube

import "encoding/json"

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

type searchResponse struct {
	Contents struct {
		SectionListRenderer struct {
			Contents []struct {
				ItemSectionRenderer struct {
					Contents []struct {
						CompactVideoRenderer struct {
							VideoID string `json:"videoId"`
							Title   struct {
								Runs []struct {
									Text string `json:"text"`
								} `json:"runs"`
							} `json:"title"`
							ByLine text `json:"shortBylineText"`
						} `json:"compactVideoRenderer"`
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
		Text string `json:"text"`
	} `json:"runs"`
}

func (t text) first() string {
	if len(t.Runs) == 0 {
		return ""
	}
	return t.Runs[0].Text
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
	return t.Thumbnails[len(t.Thumbnails)-1].URL
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
