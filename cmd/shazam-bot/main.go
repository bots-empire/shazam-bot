package main

import (
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/bots-empire/base-bot/mailing"
	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/roylee0704/gron"
	"github.com/roylee0704/gron/xtime"

	log2 "github.com/bots-empire/shazam-bot/internal/log"
	model2 "github.com/bots-empire/shazam-bot/internal/model"
	services2 "github.com/bots-empire/shazam-bot/internal/services"
	administrator2 "github.com/bots-empire/shazam-bot/internal/services/administrator"
	"github.com/bots-empire/shazam-bot/internal/services/auth"
	"github.com/bots-empire/shazam-bot/internal/utils"
)

func main() {
	rand.Seed(time.Now().Unix())

	logger := log2.NewDefaultLogger().Prefix("Shazam Bot")
	log2.PrintLogo("Shazam Bot", []string{"3C91FF"})

	model2.FillBotsConfig()
	model2.UploadAdminSettings()

	go startPrometheusHandler(logger)

	srvs := startAllBot(logger)
	model2.UploadUpdateStatistic()

	startHandlers(srvs, logger)
}

func startAllBot(log log2.Logger) []*services2.Users {
	srvs := make([]*services2.Users, 0)

	for lang, globalBot := range model2.Bots {
		startBot(globalBot, log, lang)

		service := msgs.NewService(globalBot, []int64{872383555, 1418862576, -1001683837960})

		authSrv := auth.NewAuthService(globalBot, service)
		mail := mailing.NewService(service, 100)
		adminSrv := administrator2.NewAdminService(globalBot, mail, service)
		userSrv := services2.NewUsersService(globalBot, authSrv, adminSrv, service)

		globalBot.MessageHandler = NewMessagesHandler(userSrv, adminSrv)
		globalBot.CallbackHandler = NewCallbackHandler(userSrv)
		globalBot.AdminMessageHandler = NewAdminMessagesHandler(adminSrv)
		globalBot.AdminCallBackHandler = NewAdminCallbackHandler(adminSrv)

		srvs = append(srvs, userSrv)
	}

	log.Ok("All bots is running")
	return srvs
}

func startBot(b *model2.GlobalBot, log log2.Logger, lang string) {
	var err error
	b.Bot, err = tgbotapi.NewBotAPI(b.BotToken)
	if err != nil {
		log.Fatal("%s // error start bot: %s", lang, err.Error())
	}

	u := tgbotapi.NewUpdate(0)

	b.Chanel = b.Bot.GetUpdatesChan(u)

	b.Rdb = model2.StartRedis()
	b.DataBase = model2.UploadDataBase(lang)

	b.ParseSiriTasks()
	b.ParseLangMap()
	b.ParseCommandsList()
	b.ParseAdminMap()
}

func startPrometheusHandler(logger log2.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	logger.Ok("Metrics can be read from %s port", "7011")
	metricErr := http.ListenAndServe(":7011", nil)
	if metricErr != nil {
		logger.Fatal("metrics stoped by metricErr: %s\n", metricErr.Error())
	}
}

func startHandlers(srvs []*services2.Users, logger log2.Logger) {
	wg := new(sync.WaitGroup)
	cron := gron.New()
	cron.AddFunc(gron.Every(1*xtime.Day).At("20:59"), srvs[0].SendTodayUpdateMsg)

	for _, service := range srvs {
		wg.Add(1)
		go func(handler *services2.Users, wg *sync.WaitGroup, cron *gron.Cron) {
			defer wg.Done()
			handler.ActionsWithUpdates(logger, utils.NewSpreader(time.Minute), cron)
		}(service, wg, cron)

		service.Msgs.SendNotificationToDeveloper("Bot is restarted", false)
	}

	go func() {
		time.Sleep(5 * time.Second)

		cron.Start()
	}()

	logger.Ok("All handlers are running")

	wg.Wait()
}

func NewMessagesHandler(userSrv *services2.Users, adminSrv *administrator2.Admin) *services2.MessagesHandlers {
	handle := services2.MessagesHandlers{
		Handlers: map[string]model2.Handler{},
	}

	handle.Init(userSrv, adminSrv)
	return &handle
}

func NewCallbackHandler(userSrv *services2.Users) *services2.CallBackHandlers {
	handle := services2.CallBackHandlers{
		Handlers: map[string]model2.Handler{},
	}

	handle.Init(userSrv)
	return &handle
}

func NewAdminMessagesHandler(adminSrv *administrator2.Admin) *administrator2.AdminMessagesHandlers {
	handle := administrator2.AdminMessagesHandlers{
		Handlers: map[string]model2.Handler{},
	}

	handle.Init(adminSrv)
	return &handle
}

func NewAdminCallbackHandler(adminSrv *administrator2.Admin) *administrator2.AdminCallbackHandlers {
	handle := administrator2.AdminCallbackHandlers{
		Handlers: map[string]model2.Handler{},
	}

	handle.Init(adminSrv)
	return &handle
}
