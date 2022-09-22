package model

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	adminPath      = "assets/admin"
	jsonFormatName = ".json"

	GlobalMailing = 4
	MainAdvert    = 5
)

type Admin struct {
	AdminID          map[int64]*AdminUser
	GlobalParameters map[string]*GlobalParameters
}

type GlobalParameters struct {
	Parameters        *Params
	AdvertisingChan   *AdvertChannel
	MaintenanceMode   bool
	BlockedUsers      int
	LangSelectedMap   map[string]bool
	AdvertisingText   map[int]string
	AdvertisingPhoto  map[int]string
	AdvertisingVideo  map[int]string
	AdvertisingChoice map[int]string
}

type AdminUser struct {
	Language           string
	FirstName          string
	SpecialPossibility bool
}

type Params struct {
	BonusAmount         int
	MinWithdrawalAmount int
	VoiceAmount         int
	MaxOfVoicePerDay    int
	ReferralReward      RewardsMatrix

	ButtonUnderAdvert bool

	Currency string

	TopReward []int `json:"top_reward"`
}

type AdvertChannel struct {
	Url       map[int]string `json:"url"`
	ChannelID map[int]int64  `json:"channel_id"`
}

type RewardsMatrix []RewardsLvl

func (r RewardsMatrix) GetGapByIndex(lvl, number int) *RewardsGap {
	if lvl > len(r) {
		return r[0][0]
	}

	if number > len(r[lvl-1]) {
		return r[0][0]
	}

	return r[lvl-1][number-1]
}

func (r RewardsMatrix) GetGapByCount(lvl, count int) *RewardsGap {
	level := r[lvl-1]

	for _, gap := range level {
		if gap.LeftBorder <= count && gap.RightBorder >= count {
			return gap
		}
	}

	return level[len(level)-1]
}

func (r RewardsMatrix) GetReward(lvl, count int) int {
	level := r[lvl-1]

	return level.GetReward(count)
}

func (r RewardsMatrix) MaxIndexByLvl(lvl int) int {
	level := r[lvl-1]

	return len(level)
}

func (r RewardsMatrix) MaxLevel() int {
	return len(r)
}

func (r RewardsMatrix) UpdateGap(newGap *RewardsGap) {
	level := r[newGap.Level-1]
	level.UpdateGap(newGap)
	level.Validate(newGap.Index)

	r[newGap.Level-1] = level

	SaveAdminSettings()
}

func (r RewardsMatrix) AddGap(lvl int) *RewardsGap {
	newGap := r[lvl-1].AddGap()
	SaveAdminSettings()
	return newGap
}

func (r *RewardsMatrix) AddLvl() *RewardsGap {
	*r = append(*r, newDefaultLvl(len(*r)+1))
	SaveAdminSettings()
	return r.GetGapByIndex(len(*r), 1)
}

func newDefaultLvl(lvl int) RewardsLvl {
	return RewardsLvl{
		&RewardsGap{
			LeftBorder:  1,
			RightBorder: 1,
			Amount:      1,
			Level:       lvl,
			Index:       1,
		},
	}
}

func (r *RewardsMatrix) DeleteGap(lvl, index int) *RewardsGap {
	newGap := (*r)[lvl-1].DeleteGap(index)
	(*r)[lvl-1].reindexing()
	SaveAdminSettings()
	return newGap
}

func (r *RewardsMatrix) LastGapInLvl(lvl int) bool {
	return len((*r)[lvl-1]) == 1
}

func (r *RewardsMatrix) DeleteLvl(lvl int) *RewardsGap {
	deleteLvlByIndex(r, lvl-1)
	r.reindexing()

	defer SaveAdminSettings()

	if lvl == 1 {
		return (*r)[0][0]
	}

	return (*r)[lvl-2][0]
}

func deleteLvlByIndex(r *RewardsMatrix, i int) {
	copy((*r)[i:], (*r)[i+1:])
	(*r)[len(*r)-1] = nil
	*r = (*r)[:len(*r)-1]
}

func (r *RewardsMatrix) reindexing() {
	for lvlIndex, lvl := range *r {
		for _, gap := range lvl {
			gap.Level = lvlIndex + 1
		}
	}
}

func (r *RewardsMatrix) GetLvl(lvl int) RewardsLvl {
	return (*r)[lvl-1]
}

type RewardsLvl []*RewardsGap

func (r RewardsLvl) GetGapByCount(count int) *RewardsGap {
	for _, gap := range r {
		if gap.LeftBorder >= count && gap.RightBorder <= count {
			return gap
		}
	}

	return r[len(r)-1]
}

func (r RewardsLvl) GetReward(count int) int {
	for _, gap := range r {
		if gap.LeftBorder >= count && gap.RightBorder <= count {
			return gap.Amount
		}
	}

	return r[len(r)-1].Amount
}

func (r RewardsLvl) UpdateGap(newGap *RewardsGap) {
	r[newGap.Index-1] = newGap
}

func (r *RewardsLvl) AddGap() *RewardsGap {
	lastGap := (*r)[len(*r)-1]
	newGap := &RewardsGap{
		LeftBorder:  lastGap.RightBorder + 1,
		RightBorder: lastGap.RightBorder + 1,
		Amount:      lastGap.Amount,
		Level:       lastGap.Level,
		Index:       lastGap.Index + 1,
	}

	*r = append(*r, newGap)

	return newGap
}

func (r *RewardsLvl) DeleteGap(index int) *RewardsGap {
	if index == 1 {
		deleteGapByIndex(r, index-1)
		(*r)[0].LeftBorder = 1
		return (*r)[0]
	}

	if index == len(*r) {
		deleteGapByIndex(r, index-1)
		return (*r)[index-2]
	}

	deleteGapByIndex(r, index-1)
	(*r)[index-1].LeftBorder = (*r)[index-2].RightBorder + 1

	return (*r)[index-2]
}

func deleteGapByIndex(r *RewardsLvl, i int) {
	copy((*r)[i:], (*r)[i+1:])
	(*r)[len(*r)-1] = nil
	*r = (*r)[:len(*r)-1]
}

func (r *RewardsLvl) reindexing() {
	for i, gap := range *r {
		gap.Index = i + 1
	}
}

type RewardsGap struct {
	LeftBorder  int `json:"left_border,omitempty"`
	RightBorder int `json:"right_border,omitempty"`
	Amount      int `json:"amount,omitempty"`

	Level int `json:"level"`
	Index int `json:"index"`
}

func (r *RewardsLvl) Validate(index int) {
	if index != 1 {
		r.validateLeft(index - 1)
	}

	r.validateRight(index + 1)

	for i := len(*r); i > 0; i-- {
		if r.getGapByIndex(i).RightBorder < 1 || r.getGapByIndex(i).LeftBorder < 1 {
			r.DeleteGap(i)
		}
	}

	r.reindexing()
}

func (r *RewardsLvl) validateLeft(index int) {
	gap := r.getGapByIndex(index)

	rightGap := r.getGapByIndex(index + 1)

	gap.RightBorder = rightGap.LeftBorder - 1

	if gap.LeftBorder > gap.RightBorder {
		gap.LeftBorder = gap.RightBorder
	}

	if gap.Index == 1 {
		return
	}

	r.validateLeft(index - 1)
}

func (r *RewardsLvl) validateRight(index int) {
	if index > r.getMaxIndex() {
		return
	}

	gap := r.getGapByIndex(index)

	leftGap := r.getGapByIndex(index - 1)

	gap.LeftBorder = leftGap.RightBorder + 1

	if gap.LeftBorder > gap.RightBorder {
		gap.RightBorder = gap.LeftBorder
	}

	r.validateRight(index + 1)
}

func (r RewardsLvl) getGapByIndex(index int) *RewardsGap {
	return r[index-1]
}

func (r RewardsLvl) getMaxIndex() int {
	return len(r)
}

var AdminSettings *Admin

func UploadAdminSettings() {
	var settings *Admin
	data, err := os.ReadFile(adminPath + jsonFormatName)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(data, &settings)
	if err != nil {
		fmt.Println(err)
	}

	for lang, globalBot := range Bots {
		nilSettings(settings, lang)
		for _, lang = range globalBot.LanguageInBot {
			nilSettings(settings, lang)
		}
	}

	AdminSettings = settings
	SaveAdminSettings()
}

func nilSettings(settings *Admin, lang string) {
	if settings.GlobalParameters == nil {
		settings.GlobalParameters = make(map[string]*GlobalParameters)
	}

	if settings.GlobalParameters[lang] == nil {
		settings.GlobalParameters[lang] = &GlobalParameters{}
	}

	if settings.GlobalParameters[lang].Parameters == nil {
		settings.GlobalParameters[lang].Parameters = &Params{}
	}

	if settings.GlobalParameters[lang].AdvertisingChan == nil {
		settings.GlobalParameters[lang].AdvertisingChan = &AdvertChannel{
			Url: map[int]string{
				1: "https://google.com",
				2: "https://google.com",
				3: "https://google.com",
				5: "https://google.com"},
			ChannelID: make(map[int]int64),
		}
	}

	if settings.GlobalParameters[lang].Parameters == nil {
		settings.GlobalParameters[lang].Parameters = &Params{
			TopReward: []int{10, 10, 10},
		}
	}

	if settings.GlobalParameters[lang].Parameters.TopReward == nil {
		settings.GlobalParameters[lang].Parameters.TopReward = []int{10, 10, 10}
	}

	if settings.GlobalParameters[lang].AdvertisingChoice == nil {
		settings.GlobalParameters[lang].AdvertisingChoice = make(map[int]string)
	}

	if settings.GlobalParameters[lang].AdvertisingText == nil {
		settings.GlobalParameters[lang].AdvertisingText = make(map[int]string)
	}

	if settings.GlobalParameters[lang].AdvertisingPhoto == nil {
		settings.GlobalParameters[lang].AdvertisingPhoto = make(map[int]string)
	}
	if settings.GlobalParameters[lang].AdvertisingVideo == nil {
		settings.GlobalParameters[lang].AdvertisingVideo = make(map[int]string)
	}
}

func SaveAdminSettings() {
	data, err := json.MarshalIndent(AdminSettings, "", "  ")
	if err != nil {
		panic(err)
	}

	if err = os.WriteFile(adminPath+jsonFormatName, data, 0600); err != nil {
		panic(err)
	}
}

// ----------------------------------------------------
//
// Update Statistic
//
// ----------------------------------------------------

type UpdateInfo struct {
	Mu      *sync.Mutex
	Counter int
	Day     int
}

var UpdateStatistic *UpdateInfo

func UploadUpdateStatistic() {
	info := &UpdateInfo{}
	info.Mu = new(sync.Mutex)
	strStatistic, err := Bots["it"].Rdb.Get("update_statistic").Result()
	if err != nil {
		UpdateStatistic = info
		return
	}

	data := strings.Split(strStatistic, "?")
	if len(data) != 2 {
		UpdateStatistic = info
		return
	}
	info.Counter, _ = strconv.Atoi(data[0])
	info.Day, _ = strconv.Atoi(data[1])
	UpdateStatistic = info
}

func SaveUpdateStatistic() {
	strStatistic := strconv.Itoa(UpdateStatistic.Counter) + "?" + strconv.Itoa(UpdateStatistic.Day)
	_, err := Bots["it"].Rdb.Set("update_statistic", strStatistic, 0).Result()
	if err != nil {
		log.Println(err)
	}
}

///////////////////
//Get Parameters
///////////////////

func (a *Admin) GetCurrency(lang string) string {
	return a.GlobalParameters[lang].Parameters.Currency
}

func (a *Admin) GetAdvertText(lang string, channel int) string {
	return a.GlobalParameters[lang].AdvertisingText[channel]
}

func (a *Admin) GetAdvertUrl(lang string, channel int) string {
	return a.GlobalParameters[lang].AdvertisingChan.Url[channel]
}

func (a *Admin) GetAdvertChannelID(lang string, channel int) int64 {
	return a.GlobalParameters[lang].AdvertisingChan.ChannelID[channel]
}

func (a *Admin) GetParams(lang string) *Params {
	return a.GlobalParameters[lang].Parameters
}

///////////////////
//Update Parameters
///////////////////

func (a *Admin) UpdateAdvertUrl(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingChan.Url[channel] = value
}

func (a *Admin) UpdateAdvertChannelID(lang string, value int64, channel int) {
	a.GlobalParameters[lang].AdvertisingChan.ChannelID[channel] = value
}

func (a *Admin) UpdateAdvertText(lang string, value string, channel int) {
	a.GlobalParameters[lang].AdvertisingText[channel] = value
}

func (a *Admin) UpdateAdvertPhoto(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingPhoto[channel] = value
}

func (a *Admin) UpdateAdvertVideo(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingVideo[channel] = value
}

func (a *Admin) UpdateAdvertChoice(lang string, channel int, value string) {
	a.GlobalParameters[lang].AdvertisingChoice[channel] = value
}

func (a *Admin) UpdateAdvertChan(lang string, newChan *AdvertChannel) {
	a.GlobalParameters[lang].AdvertisingChan = newChan
}

func (a *Admin) UpdateCurrency(lang string, value string) {
	a.GlobalParameters[lang].Parameters.Currency = value
}

func (a *Admin) UpdateBonusAmount(lang string, value int) {
	a.GlobalParameters[lang].Parameters.BonusAmount = value
}

func (a *Admin) UpdateMinWithdrawalAmount(lang string, value int) {
	a.GlobalParameters[lang].Parameters.MinWithdrawalAmount = value
}

func (a *Admin) UpdateVoiceAmount(lang string, value int) {
	a.GlobalParameters[lang].Parameters.VoiceAmount = value
}

func (a *Admin) UpdateMaxOfVoicePerDay(lang string, value int) {
	a.GlobalParameters[lang].Parameters.MaxOfVoicePerDay = value
}

func (a *Admin) UnderMaintenance(lang string) bool {
	return a.GlobalParameters[lang].MaintenanceMode
}
