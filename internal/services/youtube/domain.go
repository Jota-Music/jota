package youtube

type Video struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author,omitempty"`
	ChannelId string `json:"channelId,omitempty"`
	Duration  int    `json:"duration,omitempty"`
}
