// 客服工单协作平台后端入口。
//
//	server                  启动 HTTP 服务
//	server -migrate up      执行内嵌迁移后退出
//	server -migrate down    回滚全部迁移后退出
//	server -migrate reset   回滚全部再重新迁移（恢复演示数据）
//
// 配置来自环境变量，启动时会尝试读取 .env 和 ../.env（后者对应 go -C backend 的工作目录），
// 文件不存在时忽略，已存在的系统环境变量优先。
// 启动前须先 -migrate up；/api/health 以外的接口都要求 X-User-ID。
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"ticketdesk/internal/httpapi"
	"ticketdesk/internal/service"
	"ticketdesk/internal/store"
	"ticketdesk/migrations"
)

// 默认只监听回环地址：避免 Windows 每次 go run 生成新 exe 时弹出防火墙提示。
const defaultAddr = "127.0.0.1:8080"

func main() {
	loadEnv()

	migrate := flag.String("migrate", "", "执行迁移后退出：up、down 或 reset")
	flag.Parse()

	if *migrate != "" {
		runMigrate(*migrate)
		return
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL 未设置：请先运行 npm run setup 生成 .env")
	}

	st, err := store.Open(context.Background(), dsn)
	if err != nil {
		log.Fatalf("连接数据库失败（先运行 npm run db:up 和 npm run migrate）: %v", err)
	}
	defer st.Close()

	handler := httpapi.NewServer(service.New(st, nil))
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

// loadEnv 依次尝试 .env 和 ../.env。godotenv.Load 不会覆盖已存在的环境变量，
// 所以系统环境变量优先；逐个文件加载，其中一个不存在不影响另一个。
func loadEnv() {
	for _, name := range []string{".env", "../.env"} {
		_ = godotenv.Load(name)
	}
}

func runMigrate(direction string) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL 未设置：请先运行 npm run setup 生成 .env")
	}

	var err error
	switch direction {
	case "up":
		err = migrations.Up(dsn)
	case "down":
		err = migrations.Down(dsn)
	case "reset":
		err = migrations.Reset(dsn)
	default:
		log.Fatalf("-migrate 只接受 up、down 或 reset，收到 %q", direction)
	}
	if err != nil {
		log.Fatalf("migrate %s 失败: %v", direction, err)
	}
	log.Printf("migrate %s 完成", direction)
}
