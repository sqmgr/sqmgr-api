/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr/internal/database"
	"github.com/weters/sqmgr/internal/model"

	_ "github.com/lib/pq"
)

type WordList struct {
	pointer int
	words   []string
}

var seed = flag.Int64("seed", time.Now().UnixNano(), "seed value")
var numAccounts = flag.Int("accounts", 5, "number of test accounts to create")
var numSquares = flag.Int("squares", 250, "number of squares to generate")
var chance = flag.Int("chance", 50, "percent change to join a square")

func main() {
	flag.Parse()

	logrus.WithField("seed", *seed).Info("seeding random number generator")
	rand.Seed(*seed)
	words := loadWords()

	db, err := database.Open()
	if err != nil {
		panic(err)
	}
	m := model.New(db)

	accounts := make([]*model.User, *numAccounts)
	for i, _ := range accounts {
		email := fmt.Sprintf("user%d@sqmgr.com", i)
		logrus.WithField("email", email).Info("creating user")
		user, err := m.NewUser(email, "test-password")
		if err != nil {
			if err != model.ErrUserExists {
				logrus.WithError(err).Fatal("cannot create user")
			}

			logrus.WithField("email", email).Warn("user already exists")

			user, err = m.UserByEmail(email, true)
			if err != nil {
				logrus.WithError(err).Fatal("cannot load user")
			}
		}

		user.State = model.Active
		if err := user.Save(); err != nil {
			panic(err)
		}

		accounts[i] = user
	}

	for i := 0; i < *numSquares; i++ {
		user := accounts[rand.Intn(len(accounts))]

		st := model.SquaresTypeStd100
		if rand.Intn(2) == 0 {
			st = model.SquaresTypeStd25
		}

		name := words.Create(2, " ")
		logrus.WithFields(logrus.Fields{"name": name, "user": user.Email}).Info("creating squares")
		squares, err := m.NewSquares(user.ID, name, st, "joinpw")
		if err != nil {
			panic(err)
		}

		for _, account := range accounts {
			if user == account {
				continue
			}

			if rand.Intn(100) < *chance {
				logrus.WithFields(logrus.Fields{"name": name, "user": account.Email}).Info("joining squares")
				if err := account.JoinSquares(context.Background(), squares); err != nil {
					logrus.WithError(err).Fatal("could not join squares")
				}
			}
		}
	}
}

func (w *WordList) Create(nWords int, sep string) string {
	words := make([]string, nWords)
	for i, _ := range words {
		word := w.words[w.pointer]
		word = strings.ToUpper(string(word[0])) + string(word[1:])
		words[i] = word
		w.pointer++
	}

	return strings.Join(words, sep)
}

func loadWords() WordList {
	file, err := os.Open("/usr/share/dict/words")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	w := make([]string, 0)
	for scanner.Scan() {
		w = append(w, strings.ToLower(scanner.Text()))
	}

	for i := len(w) - 1; i > 0; i-- {
		n := rand.Intn(i + 1)
		w[i], w[n] = w[n], w[i]
	}

	return WordList{words: w}
}
