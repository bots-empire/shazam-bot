package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/bots-empire/shazam-bot/cfg"
	"github.com/bots-empire/shazam-bot/db/local"
)

const (
	tokensPath       = "./cfg/tokens.json"
	dbDriver         = "postgres"
	redisDefaultAddr = "127.0.0.1:6379"

	statusDeleted = "deleted"
)

var Bots = make(map[string]*GlobalBot)

type GlobalBot struct {
	BotLang string `json:"bot_lang"`

	Bot      *tgbotapi.BotAPI
	Chanel   tgbotapi.UpdatesChannel
	Rdb      *redis.Client
	DataBase *sql.DB

	MessageHandler  GlobalHandlers
	CallbackHandler GlobalHandlers

	AdminMessageHandler  GlobalHandlers
	AdminCallBackHandler GlobalHandlers

	Commands     map[string]string
	Language     map[string]map[string]string
	AdminLibrary map[string]map[string]string

	BotToken      string   `json:"bot_token"`
	BotLink       string   `json:"bot_link"`
	LanguageInBot []string `json:"language_in_bot"`
	AssistName    string   `json:"assist_name"`
}

type GlobalHandlers interface {
	GetHandler(command string) Handler
}

type Handler func(situation *Situation) error

func UploadDataBase(dbLang string) *sql.DB {
	//dataBase, err := sql.Open(dbDriver, fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
	//	"localhost", 6543, "shazam-root", "shazam-root-db")) //TODO: refactor
	//if err != nil {
	//	log.Fatalf("Failed open database: %s\n", err.Error())
	//}

	dataBase, err := sql.Open(dbDriver, fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
		"localhost", 6543, "shazam-root", "shazam-root-db")) //TODO: refactor
	if err != nil {
		log.Fatalf("Failed open database: %s\n", err.Error())
	}

	dataBase.Exec("CREATE DATABASE " + cfg.DBCfg.Names[dbLang] + ";")
	if err := dataBase.Close(); err != nil {
		log.Fatalf("Failed close database: %s\n", err.Error())
	}

	dataBase, err = sql.Open(dbDriver, fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
		"localhost", 6543, "shazam-root", cfg.DBCfg.Names[dbLang])) //TODO: refactor
	if err != nil {
		log.Fatalf("Failed open database: %s\n", err.Error())
	}

	dataBase.SetMaxOpenConns(10)
	dataBase.SetConnMaxIdleTime(30 * time.Second)

	goose.SetBaseFS(local.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err := goose.Up(dataBase, "migrations"); err != nil {
		panic(err)
	}

	err = dataBase.Ping()
	if err != nil {
		log.Fatalf("Failed upload database: %s\n", err.Error())
	}

	return dataBase
}

func StartRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisDefaultAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return rdb
}

func GetDB(botLang string) *sql.DB {
	return Bots[botLang].DataBase
}

func FillBotsConfig() {
	bytes, err := os.ReadFile(tokensPath)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(bytes, &Bots)
	if err != nil {
		panic(err)
	}

	for lang, bot := range Bots {
		bot.BotLang = lang
	}
}

func GetGlobalBot(botLang string) *GlobalBot {
	return Bots[botLang]
}

func (b *GlobalBot) GetRelationName() string {
	return "shazam.users"
}

func (b *GlobalBot) GetBotLang() string {
	return b.BotLang
}

func (b *GlobalBot) GetBot() *tgbotapi.BotAPI {
	return b.Bot
}

func (b *GlobalBot) GetDataBase() *sql.DB {
	return b.DataBase
}

func (b *GlobalBot) AvailableLang() []string {
	return b.LanguageInBot
}

func (b *GlobalBot) GetCurrency() string {
	return AdminSettings.GetCurrency(b.BotLang)
}

func (b *GlobalBot) LangText(lang, key string, values ...interface{}) string {
	formatText := b.Language[lang][key]
	return fmt.Sprintf(formatText, values...)
}

func (b *GlobalBot) GetTexts(lang string) map[string]string {
	return b.Language[lang]
}

func (b *GlobalBot) CheckAdmin(userID int64) bool {
	_, exist := AdminSettings.AdminID[userID]
	return exist
}

func (b *GlobalBot) AdminLang(userID int64) string {
	return AdminSettings.AdminID[userID].Language
}

func (b *GlobalBot) AdminText(adminLang, key string) string {
	return b.AdminLibrary[adminLang][key]
}

func (b *GlobalBot) UpdateBlockedUsers(channel int) {
}

func (b *GlobalBot) GetAdvertURL(botLang string, channel int) string {
	return AdminSettings.GetAdvertUrl(botLang, channel)
}

func (b *GlobalBot) GetAdvertText(userLang string, channel int) string {
	return AdminSettings.GetAdvertText(userLang, channel)
}

func (b *GlobalBot) GetAdvertisingPhoto(lang string, channel int) string {
	return AdminSettings.GlobalParameters[lang].AdvertisingPhoto[channel]
}

func (b *GlobalBot) GetAdvertisingVideo(lang string, channel int) string {
	return AdminSettings.GlobalParameters[lang].AdvertisingVideo[channel]
}

func (b *GlobalBot) ButtonUnderAdvert() bool {
	return AdminSettings.GlobalParameters[b.BotLang].Parameters.ButtonUnderAdvert
}

func (b *GlobalBot) AdvertisingChoice(channel int) string {
	return AdminSettings.GlobalParameters[b.BotLang].AdvertisingChoice[channel]
}

func (b *GlobalBot) BlockUser(userID int64) error {
	_, err := b.GetDataBase().Exec(`
UPDATE shazam.users
	SET status = $1
WHERE id = $2`,
		statusDeleted,
		userID)

	return errors.Wrap(err, "failed block user")
}

func (b *GlobalBot) GetMetrics(metricKey string) *prometheus.CounterVec {
	metricsByKey := map[string]*prometheus.CounterVec{
		"total_mailing_users": MailToUser,
		"total_block_users":   BlockUser,
	}

	return metricsByKey[metricKey]
}
