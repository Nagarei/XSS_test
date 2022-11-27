package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

func getEnv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}

func adminBotDo() {
	// ここでヘッドレスブラウザを立ち上げ。オプション指定無し = デフォルトの設定例。
	url := getEnv("APP_URI", "http://localhost:3000")
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// loginページ
	page := browser.MustPage(url + "/login").MustWaitLoad()

	// login試行
	// 毎回ログインするのでセッションの数が多くなりそうだけど使い捨てなので気にしないことにする
	fmt.Println("login試行...")
	page.MustElement("input").MustInput("supersecurepassword").MustType(input.Enter)
	page.MustWaitLoad()

	//コメントを承認
	fmt.Println("コメントを承認...")
	clicked := true
	for clicked {
		clicked = false
		for _, button := range page.MustElements("button") {
			fmt.Println("approve")
			button.MustClick()
			page.MustWaitLoad()
			clicked = true
			break
		}
	}

	//scriptの実行待ち
	time.Sleep(time.Second * 10)
}
func adminBot() {
	for {
		adminBotDo()
		fmt.Println("待機...")
	}
}

func main() {
	adminBot()
}
