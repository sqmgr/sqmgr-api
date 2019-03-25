package pwned

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var PwnedPasswordAPI = "https://api.pwnedpasswords.com/range"

// Client is an http client with a default timeout of one second
var Client = &http.Client{
	Timeout: time.Second,
}

var ErrInvalidResponse = errors.New("error: invalid response detected from pwnedpasswords.com")

func init() {
	if api := os.Getenv("PWNED_API"); api != "" {
		PwnedPasswordAPI = api
	}
}

func Count(password string) (int, error) {
	sha := sha1.Sum([]byte(password))
	shaHex := hex.EncodeToString(sha[:])

	prefixBytes, suffixBytes := shaHex[0:5], shaHex[5:]
	suffix := strings.ToUpper(string(suffixBytes))

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", PwnedPasswordAPI, prefixBytes), nil)
	if err != nil {
		return 0, err
	}

	res, err := Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), ":")
		if len(line) != 2 {
			return 0, ErrInvalidResponse
		}

		pwHex := strings.ToUpper(line[0])
		count, _ := strconv.Atoi(line[1])

		if pwHex == suffix {
			return count, nil
		}
	}

	return 0, nil
}
