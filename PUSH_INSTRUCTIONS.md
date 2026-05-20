# How to push this to your GitHub

The whole project sits in `notification-system-go/`. Copy it to your local machine and push it to GitHub as a new repo.

## Steps (5 minutes)

```bash
# 1. Copy the folder somewhere on your machine
cp -r /path/to/notification-system-go ~/code/notification-system-go
cd ~/code/notification-system-go

# 2. Initialize git
git init
git add .
git commit -m "feat: initial scaffold — REST API, Redis queue, worker pool, channel adapters, observability"

# 3. Create the repo on GitHub (https://github.com/new) named "notification-system-go"
#    - Description: "Horizontally scalable, multi-channel notification service in Go — Redis Streams queue, worker pool, retries, observability"
#    - Public
#    - Do NOT add README / .gitignore / license (already in repo)

# 4. Push
git remote add origin git@github.com:JD1359/notification-system-go.git
git branch -M main
git push -u origin main
```

## Verify it works locally before publishing

```bash
# Requires: docker, docker-compose
make up

# Wait ~10s for Postgres to be healthy, then:
curl -X POST http://localhost:8080/v1/notifications \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: test-001' \
  -d '{"channel":"email","to":"a@b.com","subject":"hi","body":"hello"}'

# Should return: {"id":"test-001","status":"queued",...}

curl http://localhost:8080/v1/notifications/test-001
# Should return: {"id":"test-001","status":"delivered",...}  (within 1s)

make down
```



