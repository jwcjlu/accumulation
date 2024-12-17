package nnet

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

func GetLocalIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("not found the host ip")
}

// GetValidInterfaces 获取有效网卡
func GetValidInterfaces() ([]net.Interface, error) {
	interfaces, err := net.Interfaces()
	var result []net.Interface
	if err != nil {
		return nil, nil
	}
	for _, iface := range interfaces {
		// 排除掉回环接口和无效接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		iface.Addrs()
		result = append(result, iface)
	}
	return result, nil
}
func GetClientIP(r *http.Request) (ip string) {
	// X-Forwarded-For 是一个逗号分隔的列表，第一个通常是客户端的原始IP
	defer func() {
		if len(ip) > 0 {
			host, _, err := net.SplitHostPort(ip)
			if err != nil {
				return
			}
			ip = host
		}
	}()
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// X-Real-IP 通常包含客户端的原始IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Forwarded 头字段是一个更复杂的标准，这里简单处理
	forwarded := r.Header.Get("Forwarded")
	if forwarded != "" {
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			if strings.HasPrefix(strings.ToLower(part), "for=") {
				ip := strings.TrimPrefix(part, "for=")
				ip = strings.TrimSpace(ip)
				return ip
			}
		}
	}
	// 如果上述头字段都没有，则使用RemoteAddr作为备选
	return r.RemoteAddr
}
