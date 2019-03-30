module github.com/weters/sqmgr

require (
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/sessions v1.1.3
	github.com/keegancsmith/rpc v1.1.0 // indirect
	github.com/lib/pq v1.0.0
	github.com/onsi/gomega v1.5.0
	github.com/sirupsen/logrus v1.4.0
	github.com/stamblerre/gocode v0.0.0-20190213022308-8cc90faaf476 // indirect
	github.com/synacor/argon2id v0.0.0-20190318165710-18569dfc600b
	github.com/weters/pwned v0.0.0-20190217152429-1a03bf606e34
	golang.org/x/crypto v0.0.0-20190228161510-8dd112bcdc25
	golang.org/x/tools v0.0.0-20190214204934-8dcb7bc8c7fe // indirect
)

replace github.com/weters/pwned => ../pwned
