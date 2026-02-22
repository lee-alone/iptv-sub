package utils

import (
	"fmt"
	"net"
	"strings"
)

// GetLocalIP 获取本机的真实 IPv4 地址
// 遍历所有网络接口，跳过回环接口和未启用的接口
func GetLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	for _, iface := range interfaces {
		// 跳过未启用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ipv4 := ipNet.IP.To4()
			if ipv4 == nil {
				continue // 跳过 IPv6
			}

			// 跳过 0.0.0.0
			if ipv4.String() == "0.0.0.0" {
				continue
			}

			return ipv4.String()
		}
	}

	return "127.0.0.1"
}

// GetBestLocalIP 获取最佳的本机 IP（多网卡环境下的智能选择）
// 优先选择非 192.168.x.x 的地址（可能是公网 IP）
// 其次选择 192.168.x.x
// 最后才用 127.0.0.1
func GetBestLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	var publicIP, privateIP, loopbackIP string

	for _, iface := range interfaces {
		// 跳过未启用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ipv4 := ipNet.IP.To4()
			if ipv4 == nil {
				continue // 跳过 IPv6
			}

			ipStr := ipv4.String()

			// 跳过 0.0.0.0
			if ipStr == "0.0.0.0" {
				continue
			}

			// 分类 IP
			if ipv4.IsLoopback() {
				if loopbackIP == "" {
					loopbackIP = ipStr
				}
			} else if isPrivateIP(ipv4) {
				if privateIP == "" {
					privateIP = ipStr
				}
			} else {
				if publicIP == "" {
					publicIP = ipStr
				}
			}
		}
	}

	// 优先级：公网 IP > 私网 IP > 回环 IP
	if publicIP != "" {
		return publicIP
	}
	if privateIP != "" {
		return privateIP
	}
	if loopbackIP != "" {
		return loopbackIP
	}

	return "127.0.0.1"
}

// isPrivateIP 检查 IP 是否是私网地址
func isPrivateIP(ip net.IP) bool {
	// 检查 RFC1918 私网范围
	// 10.0.0.0/8
	if ip[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}
	return false
}

// GetPrimaryAddress 构造完整的服务器地址
func GetPrimaryAddress(ip string, port int) string {
	if ip == "" {
		ip = GetBestLocalIP()
	}
	return fmt.Sprintf("http://%s:%d", ip, port)
}

// GetPlaylistURL 获取播放列表 URL
func GetPlaylistURL(ip string, port int, path string) string {
	if path == "" {
		path = "/playlist.m3u"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return GetPrimaryAddress(ip, port) + path
}
