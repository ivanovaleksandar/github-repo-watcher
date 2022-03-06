package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Repo struct {
	FullName  string `json:"full_name"`
	CreatedAt string `json:"created_at"`
}

type Repos []Repo

// Returns a list of all repositories of a user going through the paginated results from users GitHub API
func getAllRepos(user string) Repos {
	reposPerPage := 100
	githubRepoUrl := "https://api.github.com/users/" + user + "/repos?per_page=" + strconv.Itoa(reposPerPage)

	var totalResults Repos
	for page := 0; ; page++ {
		res, err := http.Get(githubRepoUrl + "&" + strconv.Itoa(page))
		if err != nil {
			log.Fatal(err)
		}
		body, err := ioutil.ReadAll(res.Body)
		defer res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		if res.StatusCode != 200 {
			log.Fatal(res.StatusCode)
		}
		var repos []Repo
		err = json.Unmarshal([]byte(body), &repos)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			totalResults = append(totalResults, repo)
		}
		if len(repos) < reposPerPage {
			break
		}
	}
	return totalResults
}

// Saves new repository in the form of 'full_name' as key and 'created_at' as a value
// Also, notifies if there is a repo created
func (repo *Repo) saveRepo(db *bolt.DB, bucketName string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		r := b.Get([]byte(repo.FullName))
		if r == nil {
			err := repo.notify()
			if err != nil {
				return err
			}
			return b.Put([]byte(repo.FullName), []byte(repo.CreatedAt))
		}
		return nil
	})
	return err
}

// Notifiies when there is a new repository created
func (repo *Repo) notify() error {
	_, err := fmt.Println(repo.CreatedAt, "repository", repo.FullName, "created")
	if err != nil {
		return err
	}
	return nil
}

func main() {
	user := flag.String("u", "", "GitHub username")
	interval := time.Duration(*flag.Int("i", 20, "Interval for checking in seconds")) * time.Second
	flag.Parse()

	db, err := bolt.Open("repos.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bucketName := *user
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan bool)

repoCheck:
	for {
		select {
		case <-quit:
			break repoCheck
		default:
			repos := getAllRepos(*user)

			for _, r := range repos {
				r.saveRepo(db, bucketName)
			}
		}
		time.Sleep(interval)
	}
}
