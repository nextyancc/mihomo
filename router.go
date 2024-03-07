package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
	"github.com/robfig/cron/v3"
)

const subconverterBaseURL = "https://subconverter.speedupvpn.com/sub"

var router *gin.Engine

// var DownloadDestFile = "static/config.yaml"
var configFile = "config1.yaml"

func init() {
	C.SetConfig(configFile)
	go startGetNodesCron() // 定时获取节点
	router = gin.Default()
	router.StaticFile("/config1.yaml", "./config1.yaml")
}
func startGetNodesCron() {
	crontab := cron.New(cron.WithSeconds())     //精确到秒
	crontab.AddFunc("0 55 */1 * * ?", getNodes) //最后一个字段 0–6 or SUN–SAT
	// crontab.AddFunc("0 5 0 * * ?", startBackup)              //数据备份
	crontab.Start()
	defer crontab.Stop()
	select {} //阻塞主线程停止
}
func getNodes() {
	var urls = []string{
		"https://raw.githubusercontent.com/Pawdroid/Free-servers/main/sub",
		"https://gist.githubusercontent.com/yewuque15/57292b943f8d808c4cd638e5edc99d54/raw",
		"https://raw.githubusercontent.com/rxsweet/proxies/main/sub/free64.txt",
		"https://gitlab.com/api/v4/projects/39360507/repository/files/data%2Fclash%2Fyaney.yaml/raw?ref=main&private_token=glpat-_xG7s-sYJPRDPgKxAk-c",
		"https://raw.githubusercontent.com/aiboboxx/v2rayfree/main/v2",
		"https://raw.githubusercontent.com/w1770946466/Auto_proxy/main/Long_term_subscription1.yaml",
		"https://raw.githubusercontent.com/w1770946466/Auto_proxy/main/Long_term_subscription2.yaml",
		"https://raw.githubusercontent.com/w1770946466/Auto_proxy/main/Long_term_subscription3.yaml",
		"https://raw.githubusercontent.com/peasoft/NoMoreWalls/master/list.txt",
		"https://raw.githubusercontent.com/chengaopan/AutoMergePublicNodes/master/list.txt",
	}

	composedURL := composeURL(urls)
	downloadedContent, err := downloadURL(composedURL)
	if err != nil {
		fmt.Printf("Error downloading content: %v\n", err)
		return
	}

	err = os.WriteFile(configFile, []byte(downloadedContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	var i int
	for {
		if _, err := executor.Parse(); err != nil {
			log.Errorln(err.Error())
			// fmt.Println(err.Error())
			fmt.Printf("configuration file %s test failed\n", C.Path.Config())
			lineNumber, parseErr := parseFailedProxyLine(err.Error())
			if parseErr != nil {
				fmt.Printf("Error parsing failed proxy: %v\n", parseErr)
				return
			}
			fmt.Printf("找到错误节点, 节点所在行: %d 开始删除节点...\n", lineNumber+2)
			i++
			err = removeLineFromFile(configFile, lineNumber+2)
			if err != nil {
				fmt.Printf("Error removing proxy line: %v\n", err)
			}
			continue
		}

		input, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Printf("读取文件出错: %v\n", err)
		}
		lines := strings.Split(string(input), "\n")
		fmt.Printf("完成删除 %d 个错误节点, 总节点数 %d 测试通过, 测试结果如下：\n", i, len(lines)-1)
		fmt.Printf("configuration file %s test is successful\n", C.Path.Config())
		break
	}

}

func Listen(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}

func composeURL(urls []string) string {
	jointURL := strings.Join(urls, "|")
	encodedURL := url.QueryEscape(jointURL)
	composedURL := fmt.Sprintf("%s?target=clash&url=%s&insert=false&emoji=true&list=true&tfo=false&scv=false&fdn=true&sort=false&udp=true&new_name=true", subconverterBaseURL, encodedURL)
	fmt.Println("订阅链接转clash链接合集如下:\n", composedURL)
	return composedURL
}

func downloadURL(targetURL string) (string, error) {
	resp, err := http.Get(targetURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseFailedProxyLine(output string) (int, error) {
	re := regexp.MustCompile(`proxy (\d+):`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("failed to find failed proxy line number")
	}
	lineNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	return lineNumber - 1, nil
}

func removeLineFromFile(filePath string, lineNumber int) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	if lineNumber < 0 || lineNumber >= len(lines) {
		return fmt.Errorf("line number out of range")
	}

	lines = append(lines[:lineNumber], lines[lineNumber+1:]...)
	output := strings.Join(lines, "\n")
	return os.WriteFile(filePath, []byte(output), 0644)
}
