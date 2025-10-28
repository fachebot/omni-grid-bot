package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fachebot/perp-dex-grid-bot/internal/config"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot"
)

var (
	version     = "dev"
	showVersion = flag.Bool("version", false, "显示版本信息")
	configFile  = flag.String("f", "etc/config.yaml", "the config file")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version)
		return
	}

	// 读取配置文件
	c, err := config.LoadFromFile(*configFile)
	if err != nil {
		logger.Fatalf("读取配置文件失败, %s", err)
	}

	// 创建数据目录
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			logger.Fatalf("创建数据目录失败, %s", err)
		}
	}

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c)

	// 运行机器人服务
	botService := telebot.NewTeleBot(svcCtx)
	if err != nil {
		logger.Fatalf("创建机器人服务失败, %s", err)
	}
	botService.Start()

	// 等待程序退出
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	botService.Stop()

	svcCtx.Close()
	logger.Infof("服务已停止")
}
