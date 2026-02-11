package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func main() {
	// 寻找串口
	targetPortName, err := findPortByDescription("USB MD AT Port")
	if err != nil {
		log.Fatalf("查找串口失败: %v", err)
	}
	fmt.Printf("找到目标串口: %s\n", targetPortName)

	// 发送 AT 指令并获取 IP
	ipAddress, err := executeATAndGetIP(targetPortName)
	if err != nil {
		log.Fatalf("AT 指令执行或解析失败: %v", err)
	}
	fmt.Printf("解析到的 IPv4 地址: %s\n", ipAddress)

	// 计算网关
	gateway := calculateGateway(ipAddress)
	fmt.Printf("计算出的网关: %s\n", gateway)

	// 根据描述找到网卡连接名称 (Interface Name)
	targetDesc := "Remote NDIS based Internet Sharing Device"
	interfaceName, err := getInterfaceNameByDescription(targetDesc)
	if err != nil {
		log.Fatalf("未找到描述为 '%s' 的网卡: %v", targetDesc, err)
	}
	fmt.Printf("网卡连接名称为: %s\n", interfaceName)

	// 设置 Windows 网络
	err = setStaticIP(interfaceName, ipAddress, gateway)
	if err != nil {
		log.Fatalf("设置网络配置失败: %v", err)
	}

	fmt.Println("------------------------------------------------")
	fmt.Println("🎉 网络配置全部成功！")
	fmt.Printf("IP: %s\n", ipAddress)
	fmt.Printf("Mask: 255.255.255.0\n")
	fmt.Printf("Gateway: %s\n", gateway)
	fmt.Printf("DNS: 223.5.5.5 / 223.6.6.6\n")
}

// GbkToUtf8 将 GBK 编码的字节流转换为 UTF-8 字符串
func GbkToUtf8(s []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func findPortByDescription(keyword string) (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", err
	}
	for _, port := range ports {
		if strings.Contains(port.Product, keyword) || strings.Contains(port.Name, keyword) {
			return port.Name, nil
		}
	}
	return "", fmt.Errorf("未找到包含 '%s' 的串口", keyword)
}

func executeATAndGetIP(portName string) (string, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return "", err
	}
	defer func(port serial.Port) {
		err := port.Close()
		if err != nil {
			log.Fatalf("关闭串口失败: %v", err)
		}
	}(port)

	commands := []string{
		"ATI",
		"AT+CGDCONT=3,\"IPV4V6\",\"CBNET\"",
		"AT+CGACT=1,3",
		"AT+CGPADDR=3",
	}

	var lastOutput string

	for _, cmd := range commands {
		fmt.Printf("-> 发送: %s\n", cmd)
		_, err := port.Write([]byte(cmd + "\r\n"))
		if err != nil {
			return "", err
		}

		time.Sleep(500 * time.Millisecond)

		buf := make([]byte, 4096)
		n, err := port.Read(buf)
		if err != nil {
			return "", err
		}
		output := string(buf[:n])

		fmt.Printf("   ... (共 %d 字节)\n", n)

		if strings.Contains(cmd, "AT+CGPADDR=3") {
			lastOutput = output
		}
	}

	re := regexp.MustCompile(`\+CGPADDR:\s*\d+,"([^"]+)"`)
	matches := re.FindStringSubmatch(lastOutput)
	if len(matches) < 2 {
		return "", fmt.Errorf("无法从响应中解析 IP地址. 响应内容: %s", lastOutput)
	}

	return matches[1], nil
}

func calculateGateway(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	parts[3] = "1"
	return strings.Join(parts, ".")
}

func getInterfaceNameByDescription(descKeyword string) (string, error) {
	psCmd := fmt.Sprintf(`Get-NetAdapter | Where-Object { $_.InterfaceDescription -match '%s' } | Select-Object -ExpandProperty Name`, descKeyword)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	decodedOutput, err := GbkToUtf8(outputBytes)
	if err != nil {
		return "", fmt.Errorf("解码 PowerShell 输出失败: %v", err)
	}

	name := strings.TrimSpace(decodedOutput)
	if name == "" {
		return "", fmt.Errorf("PowerShell 未返回任何网卡名称，请检查设备描述符是否匹配")
	}
	return name, nil
}

func setStaticIP(interfaceName, ip, gateway string) error {
	mask := "255.255.255.0"

	fmt.Printf("正在配置网卡 '%s'...\n", interfaceName)

	// 设置 IP
	cmdIP := exec.Command("netsh", "interface", "ip", "set", "address", interfaceName, "static", ip, mask, gateway)
	if output, err := cmdIP.CombinedOutput(); err != nil {
		decodedErr, _ := GbkToUtf8(output)
		return fmt.Errorf("设置 IP 失败: %v, 输出: %s", err, decodedErr)
	}
	fmt.Println("√ IP、掩码、网关设置成功")

	// 设置主 DNS
	cmdDNS1 := exec.Command("netsh", "interface", "ip", "set", "dns", interfaceName, "static", "223.5.5.5")
	if output, err := cmdDNS1.CombinedOutput(); err != nil {
		decodedErr, _ := GbkToUtf8(output)
		return fmt.Errorf("设置主 DNS 失败: %v, 输出: %s", err, decodedErr)
	}
	fmt.Println("√ 主 DNS 设置成功")

	// 添加备用 DNS
	cmdDNS2 := exec.Command("netsh", "interface", "ip", "add", "dns", interfaceName, "223.6.6.6", "index=2")
	if output, err := cmdDNS2.CombinedOutput(); err != nil {
		decodedErr, _ := GbkToUtf8(output)
		fmt.Printf("警告: 添加备用 DNS 异常: %s\n", decodedErr)
	} else {
		fmt.Println("√ 备用 DNS 设置成功")
	}

	return nil
}
