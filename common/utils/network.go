package utils

import "fmt"

func PrimaryIPv4() (string, error) {
	ip, err := ExecShell(`route -n | grep ^[0-9] | grep -v docker | grep -v "169.254.0.0" | \
	awk '{print $NF}' | head -n1 | xargs -i ifconfig {} | grep inet | \
	grep netmask | grep broadcast | awk '{print $2}'`)
	if err != nil {
		return "", err
	}
	if ip == "" {
		return "", fmt.Errorf("primary IPv4 not found")
	}
	return ip, nil
}
