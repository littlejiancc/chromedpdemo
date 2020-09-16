package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const (
	_url = "https://weixin.sogou.com/weixin?p=01030402&query=%E8%85%BE%E8%AE%AF%E7%8E%84%E6%AD%A6%E5%AE%9E%E9%AA%8C%E5%AE%A4&type=1&ie=utf8"
	_ua  = "Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"
)

const (
	_linkReg  = `<p style="text-align: left;"><span style="font-size: 16px;">•&nbsp;<span style=" box-sizing: border-box; width: 100%;padding-right: 5px;padding-left: 5px;flex-basis: 0px;flex-grow: 1;max-width: 100%; ">.+?<a href=".+?" rel="nofollow" style="box-sizing: border-box;color: rgb\(0, 123, 255\);" data-linktype="2"><br style="box-sizing: border-box;">(.+?)</a></span></span></p>`
	_titleReg = `<p style="box-sizing: border-box;margin-top: 0\.25rem !important;margin-bottom: 0\.25rem !important;text-align: left;"><small style="box-sizing: border-box;font-size: 12\.8px;"><span style="font-size: 16px;">&nbsp;&nbsp;&nbsp;・</span></small><span style="font-size: 16px;">&nbsp;</span><q style="box-sizing: border-box;"><span style="font-size: 16px;">(.+?)</span>`
)

var (
	_xLinkRegexp  *regexp.Regexp
	_xTitleRegexp *regexp.Regexp
)

func init() {
	_xLinkRegexp = compileReg(_linkReg)
	_xTitleRegexp = compileReg(_titleReg)
}

func compileReg(reg string) *regexp.Regexp {
	compile, _ := regexp.Compile(reg)
	return compile
}

func main() {
	html := scrapeNewArticleHtml()
	printArticle(html)
}

// scrapeNewArticleHtml 获取最新文章内容html
func scrapeNewArticleHtml() (html string) {
	// 参数配置
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false), // 是否打开浏览器调试
		chromedp.UserAgent(_ua),          // 设置User-Agent
	}
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()

	// 创建chrome实例
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	// 设置超时时间
	ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 监听得到第二个tab页的target ID
	ch := make(chan target.ID, 1)
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*target.EventTargetCreated); ok &&
			// if OpenerID == "", this is the first tab.
			ev.TargetInfo.OpenerID != "" {
			ch <- ev.TargetInfo.TargetID
		}
	})

	var body string
	if err := chromedp.Run(ctx,
		chromedp.Tasks{
			// 打开导航
			chromedp.Navigate(_url),
			// 等待元素加载完成
			chromedp.WaitVisible("body"),
			// 延迟2秒
			chromedp.Sleep(2 * time.Second),
			// 点击事件
			chromedp.Click(`a[uigs="account_article_0"]`, chromedp.NodeVisible),
			chromedp.Sleep(3 * time.Second),
			// 获取html
			chromedp.OuterHTML("html", &body, chromedp.ByQuery),
		},
	); err != nil {
		log.Printf("[scrapeNewArticle] chromedp Run fail,err: %s", err.Error())
		return
	}
	// 第二个tab页
	newCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
	defer cancel()
	if err := chromedp.Run(
		newCtx,
		chromedp.Sleep(1*time.Second),
		chromedp.OuterHTML("#js_content", &html, chromedp.ByID),
	); err != nil {
		log.Printf("[scrapeNewArticle] chromedp Run fail,err: %s", err.Error())
		return
	}
	return html
}

// printArticle 正则获取文章并且打印
func printArticle(html string) {
	if html == "" {
		return
	}

	var titleList, linkList [][]string
	linkList = _xLinkRegexp.FindAllStringSubmatch(html, -1)
	titleList = _xTitleRegexp.FindAllStringSubmatch(html, -1)
	if len(linkList) != len(titleList) {
		return
	}
	for i := 0; i < len(linkList); i++ {
		if len(linkList[i]) > 0 && len(titleList[i]) > 0 {
			link := linkList[i][1]
			title := titleList[i][1]
			fmt.Println("link: ", link)
			fmt.Println("title: ", title)
		}
	}
}
