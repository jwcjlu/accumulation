package model

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

func TestQuery(t *testing.T) {
	traffics := Bandwidths{
		&Bandwidth{MacAddress: "00:11:22:33:44:55", Ip: "192.168.0.1", Port: "100", UpLen: 200, DownLen: 250, StartTime: time.Now().Add(-2 * time.Hour).Unix()},
		&Bandwidth{MacAddress: "11:22:33:44:55:66", Ip: "192.168.0.2", Port: "150", UpLen: 250, DownLen: 200, StartTime: time.Now().Add(-59 * time.Minute).Unix()},
		&Bandwidth{MacAddress: "22:33:44:55:66:77", Ip: "192.168.0.3", Port: "200", UpLen: 300, DownLen: 100, StartTime: time.Now().Add(-30 * time.Minute).Unix()},
		&Bandwidth{MacAddress: "33:44:55:66:77:88", Ip: "192.168.0.4", Port: "250", UpLen: 350, DownLen: 120, StartTime: time.Now().Add(-20 * time.Minute).Unix()},
		&Bandwidth{MacAddress: "44:55:66:77:88:99", Ip: "192.168.0.5", Port: "300", UpLen: 400, DownLen: 140, StartTime: time.Now().Add(-33 * time.Minute).Unix()},
		&Bandwidth{MacAddress: "44:55:66:77:88:10", Ip: "192.168.0.6", Port: "120", UpLen: 300, DownLen: 130, StartTime: time.Now().Add(-2 * time.Minute).Unix()},
		&Bandwidth{MacAddress: "44:55:66:77:88:12", Ip: "192.168.0.7", Port: "200", UpLen: 150, DownLen: 100, StartTime: time.Now().Unix()},
	}
	// 对 NetworkTraffics 进行排序
	sort.Sort(traffics)

	// 定义查询时间范围
	startTime := time.Now().Unix() - 3600 // 查询过去一个小时的数据
	nTraffics := traffics.Search(startTime, 0)
	for _, t := range nTraffics {
		t.Print()
	}
	fmt.Println("===================================")
	endTime := time.Now().Unix() - 180
	nTraffics = traffics.Search(startTime, endTime)
	for _, t := range nTraffics {
		t.Print()
	}
}
