// adminhash 生成 RelayPulse 管理员密码的 bcrypt 哈希。
// 用法：go run ./cmd/adminhash
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "管理员密码: ")
	password, err := reader.ReadString('\n')
	if err != nil && len(password) == 0 {
		fmt.Fprintf(os.Stderr, "读取密码失败: %v\n", err)
		os.Exit(1)
	}
	password = strings.TrimRight(password, "\r\n")
	if password == "" {
		fmt.Fprintln(os.Stderr, "密码不能为空")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成密码哈希失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}
