package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type User struct {
	User        string `json:"login"`
	PublicRepos int    `json:"public_repos"`
}
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

// Returns number of repositories for user
func getNumberOfRepos(user string) int {
	githubRepoUrl := "https://api.github.com/users/" + user

	var u User
	res, err := http.Get(githubRepoUrl)
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
	err = json.Unmarshal([]byte(body), &u)
	if err != nil {
		log.Fatal(err)
	}
	numberOfRepos.Add(float64(u.PublicRepos))

	return u.PublicRepos
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

var (
	numberOfRepos = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_github_repos",
			Help: "Number of repositories for user",
		})
)

func main() {
	user := os.Getenv("GITHUB_USERNAME")
	checkInterval, _ := strconv.Atoi(os.Getenv("CHECK_INTERVAL"))
	interval := time.Duration(checkInterval) * time.Second

	db, err := bolt.Open("repos.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bucketName := user
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

	go func() {
		quit := make(chan bool)

	repoCheck:
		for {
			select {
			case <-quit:
				break repoCheck
			default:
				getNumberOfRepos(user)
				repos := getAllRepos(user)

				for _, r := range repos {
					r.saveRepo(db, bucketName)
				}
			}
			time.Sleep(interval)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	prometheus.MustRegister(numberOfRepos)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
