package youtube

type Video struct {
	ID    string
	Title string
}

type Audio struct {
	Url      string `json:"url"`
	Duration int    `json:"duration"`
	ExpireAt int64  `json:"expireAt"`
}

type format struct {
	Itag     int    `json:"itag"`
	MimeType string `json:"mimeType"`
	URL      string `json:"url"`
}

type playerResponse struct {
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
