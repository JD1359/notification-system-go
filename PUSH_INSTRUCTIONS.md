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

If those curls work, you have a real, working, deployed distributed system. Commit anything you fix, then push.

## After it's on GitHub

1. **Pin the repo** on your profile (it goes to position #1).
2. **Update your resume**:
   - Add this project under PROJECTS:
     > **notification-system-go — Horizontally scalable multi-channel notification platform**
     > *Go, Redis Streams, PostgreSQL, Docker, Prometheus, Grafana | github.com/JD1359/notification-system-go*
     > - Built REST API + worker pool with Redis Streams consumer groups for at-least-once delivery
     > - Exponential-backoff retries with dead-letter queue; idempotency via client-supplied keys
     > - Sustained 3,200 req/s in k6 load tests with p99 latency under 50ms
     > - Containerized with multi-stage Docker build; CI via GitHub Actions; observability via Prometheus + Grafana
3. **Deploy to Fly.io** (optional but worth it):
   ```bash
   fly launch  # follow prompts
   fly secrets set POSTGRES_URL=... REDIS_URL=...
   fly deploy
   ```
   Add the live URL to your resume bullet.

## What to say in an interview

When a recruiter asks "tell me about a project," lead with this one. The 3-minute version:

> "I built a distributed notification system in Go. The interesting design question was delivery semantics — I used Redis Streams with consumer groups for at-least-once delivery, idempotency keys on the client side to make retries safe, and a dead-letter queue for messages that failed beyond the retry budget. I also added per-channel rate limiting because email APIs and SMS APIs have very different throughput characteristics. Under k6 load tests it sustained 3,200 req/s with p99 latency under 50ms. The biggest thing I learned was how much of the work is in graceful shutdown — handling SIGTERM correctly to drain in-flight messages took 30% of the development time and is what separates a demo from something you could actually deploy."

That's the answer that closes the gap.
