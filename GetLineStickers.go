package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const ShellToUse = "bash"

type Setting struct {
	AniamtionUrl   string `json:"animation_url"`
	SoundUrl   string `json:"sound_url"`
}

var setting Setting

var IS_ANIMATION = true
var STICKER_DIR = ""

var STATIC_DIR = "靜圖"
var ANIMATION_DIR = "動圖"
var GIF_DIR = "GIF"
var SOUND_DIR = "Sound"


func main() {

	// 讀取json
	jsonFile, err := os.Open(GetCurPath() + "/setting.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	setting = Setting{}
	json.Unmarshal(byteValue, &setting)

	// 輸入Line貼圖網址
	fmt.Print("請輸入Line貼圖網址：")
	var url string
	fmt.Scanln(&url)

	// 抓取Line貼圖 標題 靜態圖 動態圖
	lineName, staticUrls, animationUrls, soundUrls := getLineInfo(url)

	// 設定資料夾名稱
	STICKER_DIR = lineName

	// 建立資料夾
	os.MkdirAll(GetCurPath() + STICKER_DIR, 0766)
	os.MkdirAll(GetCurPath() + STICKER_DIR + "/" + STATIC_DIR, 0766)
	os.MkdirAll(GetCurPath() + STICKER_DIR + "/" + ANIMATION_DIR, 0766)
	os.MkdirAll(GetCurPath() + STICKER_DIR + "/" + GIF_DIR, 0766)
	os.MkdirAll(GetCurPath() + STICKER_DIR + "/" + SOUND_DIR, 0766)
	if STICKER_DIR != "" {
		Shellout("rm -rf " + GetCurPath() + STICKER_DIR + "/" + STATIC_DIR + "/*")
		Shellout("rm -rf " + GetCurPath() + STICKER_DIR + "/" + ANIMATION_DIR + "/*")
		Shellout("rm -rf " + GetCurPath() + STICKER_DIR + "/" + GIF_DIR + "/*")
		Shellout("rm -rf " + GetCurPath() + STICKER_DIR + "/" + SOUND_DIR + "/*")
	}

	// 下載靜圖
	downloadByUrl(staticUrls, true, STATIC_DIR)

	// 下載動圖
	downloadByUrl(animationUrls, true, ANIMATION_DIR)

	// 下載音效
	downloadByUrl(soundUrls, false, SOUND_DIR)

	waitGroup.Wait()

	// 將動圖轉成GIF
	i := 0
	for range staticUrls {
		err, _, output := Shellout(GetCurPath() + "apng2gif " + GetCurPath() + STICKER_DIR + "/" + ANIMATION_DIR + "/" + strconv.Itoa(i) + ".png")
		if err != nil {
			fmt.Println(output)
		}
		i++
	}
	if STICKER_DIR != "" {
		Shellout("mv " + GetCurPath() + STICKER_DIR + "/" + ANIMATION_DIR + "/*.gif " + GetCurPath() + STICKER_DIR + "/" + GIF_DIR)
	}

}

// 取得目前執行檔路徑
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	rst := filepath.Dir(path)
	return rst + "/"
}

// 執行系統命令
func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

// 靜態轉動態網址
func stickerToAnimation(url string) string {
	return strings.Replace(url, "ANDROID/sticker.png", setting.AniamtionUrl, -1)
}

// 靜態轉音效網址
func stickerToSound(url string) string {
	return strings.Replace(url, "ANDROID/sticker.png", setting.SoundUrl, -1)
}

// 取得Line貼圖的標題與貼圖地址
func getLineInfo(url string) (string, []string, []string, []string) {
	resp, err := http.Get(url)
	if err != nil {

	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}

	html := string(body)

	// 取得貼圖標題
	titles := strings.Split(html, "<h3 class=\"mdCMN08Ttl\">")
	title := strings.Split(titles[1], "</h3>")

	// 標題移除一些特殊字元
	lineName := strings.Replace(title[0], " ", "", -1)
	lineName = strings.Replace(lineName, "&#39;", "", -1)
	lineName = strings.Replace(lineName, ":", "", -1)
	lineName = strings.Replace(lineName, "&", "", -1)
	lineName = strings.Replace(lineName, "&", "", -1)
	lineName = strings.Replace(lineName, "#", "", -1)
	lineName = strings.Replace(lineName, ";", "", -1)
	lineName = strings.Replace(lineName, "%", "", -1)
	lineName = strings.Replace(lineName, "(", "", -1)
	lineName = strings.Replace(lineName, ")", "", -1)

	// 取得貼圖所有的url
	staticUrls := []string{}
	animationUrls := []string{}
	soundUrls := []string{}
	stickers := strings.Split(html, "background-image:url(")
	for _, sticker := range stickers {
		u := strings.Split(sticker, ";compress=true);")
		staticUrls = append(staticUrls, u[0])
		animationUrls = append(animationUrls, stickerToAnimation(u[0]))
		soundUrls = append(soundUrls, stickerToSound(u[0]))
	}

	// 移除第一個不是貼圖地址的元素
	staticUrls = append(staticUrls[:0], staticUrls[1:]...)
	animationUrls = append(animationUrls[:0], animationUrls[1:]...)
	soundUrls = append(soundUrls[:0], soundUrls[1:]...)

	fmt.Println(soundUrls)

	return lineName, staticUrls, animationUrls, soundUrls
}

// 下載多個圖片
func downloadByUrl(urls []string, isPNG bool, subDir string) {
	i := 0
	for _, url := range urls {
		waitGroup.Add(1)
		go download(url, i, isPNG, subDir)
		i++
	}
}

// 下載圖片goroutine
var waitGroup = new(sync.WaitGroup)
func download(url string, i int, isPNG bool, subDir string) {
	fmt.Printf("開始下載:%s %t\n", url, isPNG)
	res, err := http.Get(url)
	if err != nil || res.StatusCode != 200 {
		fmt.Printf("下載失敗:%s\n", res.Request.URL)
		waitGroup.Done()
		return
	}
	fmt.Printf("開始讀取文件內容,url=%s\n", url)
	data, err2 := ioutil.ReadAll(res.Body)
	if err2 != nil {
		fmt.Printf("讀取資料失敗\n")
		waitGroup.Done()
		return
	}
	if isPNG {
		ioutil.WriteFile(fmt.Sprintf(GetCurPath() + STICKER_DIR + "/" + subDir + "/%d.png", i), data, 0644)
	} else {
		ioutil.WriteFile(fmt.Sprintf(GetCurPath() + STICKER_DIR + "/" + SOUND_DIR + "/%d.m4a", i), data, 0644)
	}

	waitGroup.Done()
}