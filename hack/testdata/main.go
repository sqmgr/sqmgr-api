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
var numGrids = flag.Int("grids", 250, "number of grids to generate")
var chance = flag.Int("chance", 50, "percent change to join a grid")

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

	for i := 0; i < *numGrids; i++ {
		user := accounts[rand.Intn(len(accounts))]

		st := model.GridTypeStd100
		if rand.Intn(2) == 0 {
			st = model.GridTypeStd25
		}

		name := words.Create(2, " ")
		logrus.WithFields(logrus.Fields{"name": name, "user": user.Email}).Info("creating grid")
		grid, err := m.NewGrid(user.ID, name, st, "joinpw")
		if err != nil {
			panic(err)
		}

		homeTeam := words.Create(2, " ")
		grid.Settings().SetHomeTeamName(homeTeam)
		grid.Settings().SetHomeTeamColor1(color())
		grid.Settings().SetHomeTeamColor2(color())
		awayTeam := words.Create(2, " ")
		grid.Settings().SetAwayTeamName(awayTeam)
		grid.Settings().SetAwayTeamColor1(color())
		grid.Settings().SetAwayTeamColor2(color())
		if err := grid.Save(); err != nil {
			panic(err)
		}

		for _, account := range accounts {
			if user == account {
				continue
			}

			if rand.Intn(100) < *chance {
				logrus.WithFields(logrus.Fields{"name": name, "user": account.Email}).Info("joining grid")
				if err := account.JoinGrid(context.Background(), grid); err != nil {
					logrus.WithError(err).Fatal("could not join grid")
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

var colorList = []string{"000000", "002244", "002C5F", "00338D", "004953", "00539B", "005778", "005A8B", "006778", "0073CF", "0085CA",
	"008E97", "03202F", "0B162A", "0B2265", "203731", "22150C", "241773", "34302B",
	"4B92DB", "4F2683", "565A5C", "5B2B2F", "69BE28", "773141", "97233F", "9E7C0C", "9F792C",
	"9F8958", "A5ACAF", "A71930", "AA0000", "ACC0C6", "B0B7BC", "B1BABF", "B3995D",
	"BFC0BF", "C60C30", "C83803", "D50A0A", "D7A22A", "E31837", "E9BF9B", "F58220", "FB4F14", "FF7900", "FFB612", "FFC62F",
}

func color() string {
	return "#" + colorList[rand.Intn(len(colorList))]
}
