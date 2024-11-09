package jsonmodels

type VideoJson struct {
	Id           string `json:"id"`
	SectionStart int    `json:"sectionStart"`
	SectionEnd   int    `json:"sectionEnd"`
}

type ChannelJson struct {
	Name   string      `json:"name"`
	Videos []VideoJson `json:"videos"`
}

type ChannelsJson struct {
	Channels []ChannelJson `json:"channels"`
}

type SubmitListRequestJson struct {
	VideoListUrl string `json:"videoListUrl"`
}
