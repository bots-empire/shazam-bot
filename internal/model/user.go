package model

type User struct {
	ID             int64  `json:"id"`
	Balance        int    `json:"balance"`
	Completed      int    `json:"completed"`
	CompletedToday int    `json:"completed_today"`
	LastShazam     int64  `json:"last_shazam"`
	FatherID       int64  `json:"father_id"`
	AllReferrals   string `json:"all_referrals"` // 10/20/30/40
	AdvertChannel  int    `json:"advert_channel"`
	ReferralCount  int    `json:"referral_count"`
	TakeBonus      bool   `json:"take_bonus"`
	Language       string `json:"language"`
	Status         string `json:"status"`
}
