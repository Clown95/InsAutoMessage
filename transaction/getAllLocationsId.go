package transaction

import (
	"InsAutoMessage/config"
	DB "InsAutoMessage/database"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type City struct {
	CityList []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"city_list"`
	NextPage int    `json:"next_page"`
	Status   string `json:"status"`
}

type LocationA struct {
	LocationList []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"location_list"`
	NextPage int    `json:"next_page"`
	Status   string `json:"status"`
}

func GetHttpbyProxy(rawUrl, proxyAddr, cookie string) string {
	method := "GET"
	// 解析代理地址
	proxy, err := url.Parse(proxyAddr)
	if err != nil {
		log.Println("解析代理地址错误:", err)

	}

	// 配置网络传输
	netTransport := &http.Transport{
		Proxy:               http.ProxyURL(proxy),
		MaxIdleConnsPerHost: 10,

		ResponseHeaderTimeout: 5 * time.Second,
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: netTransport,
	}

	req, err := http.NewRequest(method, rawUrl, nil)

	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header.Add("accept", "*/*")
	req.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Add("cookie", cookie)
	req.Header.Add("priority", "u=1, i")
	req.Header.Add("sec-ch-ua", "\"Not(A:Brand\";v=\"99\", \"Google Chrome\";v=\"133\", \"Chromium\";v=\"133\"")
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36")
	req.Header.Add("x-ig-app-id", "936619743392459")

	res, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	//fmt.Println(string(body))

	return string(body)
}

type Location struct {
	LocationName string
	LocationId   string
	State        int
}

func GetlocationID(proxyAddr, cookie string) {

	//proxyAddr := "http://127.0.0.1:7890"
	//cookie := `ig_did=0202D1CC-EAE2-4A81-AEF3-ADBE97792979; ds_user_id=70375669226; dpr=1.25; sessionid=70375669226%3AyfTiHp3E75OxOL%3A5%3AAYeMbTHMhUpMsFURIszVIaqZVebhrTjuEPZejmrVhQ; mid=Z8WpHwALAAEXLObgmI3nJEdIQjqf; csrftoken=Sg9ax0M96jb3fFfpHvucoskUIJu9ZzmM; ps_l=1; ps_n=1; wd=1652x1253; datr=SN7GZyiDYtz7_E53nXaz-RQN; rur="HIL\05470375669226\0541772622288:01f7717d3aa9ee0c8290049d192b0b3032adbbcac456b0ef69c37089e824083de95d8f43"`

	cityList := make(map[string]string)
	page1 := 1
	isEnd1 := false

	LocationList := make(map[string]string)
	page2 := 1
	isEnd2 := false

	//defer db.Close()
	for true {

		rawUrl := fmt.Sprintf("https://www.instagram.com/api/v1/locations/country/directory/?directory_code=IN&page=%d", page1)

		content := GetHttpbyProxy(rawUrl, proxyAddr, cookie)
		var directory City
		err := json.Unmarshal([]byte(content), &directory)
		if err != nil {
			return
		}

		for _, city := range directory.CityList {

			if _, exists := cityList[city.Name]; exists {
				isEnd1 = true
				//break
			}

			//cityList[city.Name] = city.Id
			//	fmt.Println(city.Name, city.Id)

			for true {

				directoryUrl := fmt.Sprintf("https://www.instagram.com/api/v1/locations/city/directory/?directory_code=%s&page=%d", city.Id, page2)
				content = GetHttpbyProxy(directoryUrl, proxyAddr, cookie)

				var location LocationA
				err := json.Unmarshal([]byte(content), &location)
				if err != nil {
					return
				}

				for _, item := range location.LocationList {
					if _, exists := LocationList[item.Name]; exists {
						isEnd2 = true
					}

					line := fmt.Sprintf("%s %s", item.Name, item.Id)
					fmt.Println(line)

					loc, err := DB.GetLocationsByLocationId(config.GormDb, item.Id)
					if err != nil {
						return
					}

					if len(loc) == 0 {
						loc1 := &DB.Location{
							LocationName: item.Name,
							LocationID:   item.Id,
							Count:        0,
							State:        0,
						}

						err = DB.CreateLocation(config.GormDb, loc1)
						if err != nil {
							fmt.Println("创建失败:", err)
							continue
						}
					} else {
						fmt.Printf("%s已经存在", item.Name)
					}

				}

				if isEnd2 {
					break
				}
				page2++
			}

		}

		if isEnd1 {
			break
		}

		page1++
		time.Sleep(2 * time.Second)

	}

}
