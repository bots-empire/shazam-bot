package model

type User struct {
	ID             int64  `json:"id"`
	Balance        int    `json:"balance"`
	Completed      int    `json:"completed"`
	CompletedToday int    `json:"completed_today"`
	LastVoice      int64  `json:"last_voice"`
	AdvertChannel  int    `json:"advert_channel"`
	ReferralCount  int    `json:"referral_count"`
	TakeBonus      bool   `json:"take_bonus"`
	Language       string `json:"language"`
	Status         string `json:"status"`
}
