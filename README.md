# GitHub repo watcher

An app that watches when a repository is created for a user and notifies on console. The app retains it state even after restarts using Bolt DB.

It exposes the number if user repositories in a prometheus metrics endpoint (`/metrics`) on port `8080`.

## Build

```
go build
chmod +x github-repo-watcher
```

## Run

Before running it keep in mind that the following parameters can be changed:
```
# Name or org that should be monitored
GITHUB_USERNAME=<username>
# Seconds between each collection (the GitHub API can get rate limitted quite fast)
CHECK_INTERVAL=300
# Directory of db data
DB_PATH=/tmp/db
```

### Local

```
export GITHUB_USERNAME=<user>
export CHECK_INTERVAL=300
export DB_PATH=/tmp/db

./github-repo-watcher
```

### Dockerfile 

```
docker build -t github-repo-watcher .
docker run -e GITHUB_USERNAME=<user> -e CHECK_INTERVAL=300 -e DB_PATH=/tmp/db -p 8080:8080 -v $(pwd)/db/:/tmp/db github-repo-watcher 
```

### Helm 

```
kind create cluster --config kind.yaml
helm install my-deploy chart/ -f chart/values.local.yaml
kubectl port-forward pod/<pod-name> 8080:8080
```

# To Do

- [x] Helm chart
- [ ] Option for using authenticated calls in GitHub (to not be rate limitted immediatelly) 
- [ ] Watch for deleted repos as well
