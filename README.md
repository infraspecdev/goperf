# GoPerf

## The Problem

When you're building an API, you need to answer: _How fast is it? How many users can it handle? What happens under load?_

You could manually test by refreshing your browser. Or you could use a tool that sends thousands of requests concurrently and tells you what your users will actually experience.

Tools like [hey](https://github.com/rakyll/hey), [wrk](https://github.com/wg/wrk), and [k6](https://k6.io/) solve this problem. Your job is to build one yourself.

## What You're Building

A command-line tool that load tests HTTP APIs and reports performance metrics.

### Example Usage

```bash
$ goperf -url https://api.example.com/users -concurrency 50 -duration 30s
```

### Example Output

```
Target:     https://api.example.com/users
Duration:   30.0s
Requests:   1,523 total (1,520 succeeded, 3 failed)

Latency:
  Fastest:  12ms
  Slowest:  892ms
  Average:  45ms
  p50:      38ms
  p90:      89ms
  p99:      234ms

Throughput: 50.7 requests/sec
```

The specific CLI flags, output format, and features beyond this are yours to design.

## Why This Project

This isn't just about building a tool. By the end, you should be able to:

- Write production-quality Go code
- Make and defend design decisions
- Debug complex issues independently
- Deploy and run software in the cloud
- Explain what you built and why you built it that way

## Requirements

### Must Have (MVP by Week 4)

- [ ] Accepts a target URL and test parameters via command line
- [ ] Makes concurrent HTTP requests for a specified duration
- [ ] Reports latency statistics including percentiles (p50, p90, p99)
- [ ] Reports throughput (requests per second)
- [ ] Reports success/failure counts
- [ ] Runs on AWS EC2

### Should Have (by Week 8)

- [ ] Supports different HTTP methods (GET, POST, PUT, DELETE)
- [ ] Supports custom headers and request bodies
- [ ] Loads configuration from a file
- [ ] Stores test history for comparison
- [ ] Automated build and deployment (CI/CD)
- [ ] Documented and tested

### Could Have (stretch goals)

- [ ] Real-time progress output during tests
- [ ] HTML report generation
- [ ] Distributed load testing from multiple machines

## Constraints

- **Language:** Go (this is a Go learning project)
- **Deployment:** Must run on AWS EC2, not just your laptop
- **Timeline:** 8 weeks, with MVP checkpoint at Week 4

## Infrastructure

```
AWS Account:    interns-dev (472764165211)
VPC:            vpc-05527bfd9534046db
Access:         AWS SSO → Interns_Dev group
```

## Evaluation

You will be evaluated on:

| Criteria                    | What We're Looking For                                                                 |
| --------------------------- | -------------------------------------------------------------------------------------- |
| **Technical Understanding** | Can you explain what you built and why? Can you walk through your code?                |
| **Code Quality**            | Is it readable? Tested? Would you be comfortable handing it to someone else?           |
| **Ownership & Initiative**  | Did you make decisions or wait for instructions? Did you go beyond the minimum?        |
| **Problem-Solving**         | When stuck, did you try to unblock yourself before asking? Did you research?           |
| **Communication**           | Are your commits clear? Can you explain technical concepts? Do you ask good questions? |
| **Learning Agility**        | Did you pick up new concepts? Did you improve over the 8 weeks?                        |

### What Success Looks Like

**Week 4 (MVP):** You demo a working load tester on EC2. It makes concurrent requests, measures latency, and reports percentiles. You can explain how it works and what trade-offs you made.

**Week 8 (Final):** You demo a polished tool with additional features. The code is tested and documented. You present confidently, handle questions well, and reflect on what you learned.

## Working Agreements

- **The 3-Hour Rule:** Stuck for 3 hours on a specific error? Ask for help. Struggling to learn is good. Spinning in circles is not.

- **Own Your Decisions:** You'll need to make design choices. Make them, document why, and be ready to discuss. There's rarely one right answer.

- **Demo on EC2:** Every Friday, show what you built running in the cloud. "It works on my machine" doesn't count.

- **Communicate Early:** If you're blocked, confused, or going to miss a deadline — say so early. Silence is the only wrong answer.

## Standards

### Communication (Google Chat)

- Respond within 2 hours during working hours
- Post daily updates: what you did, what's next, any blockers
- Document decisions in writing

### Workflow

- **Monday:** Sprint planning / sync with mentor
- **Mon-Fri:** Build. Async communication for blockers.
- **Friday:** Demo and review sync

### Code

- Tests exist and pass
- Code passes `go vet` (at minimum)
- PRs are reviewable chunks, not massive dumps
- Commits explain _why_, not just _what_

## Getting Started

1. Install Go 1.25+ and Docker/Podman
2. Study existing load testing tools (`hey` is a good starting point — it's written in Go)
3. Understand the problem before writing code. Ask questions, discuss with your teammate and mentor, and make sure you have clarity on what you're building and why.

## Team

| Role   | Name           |
| ------ | -------------- |
| Mentor | Rajat          |
| Buddy  | Rahul          |
| Intern | Eshaan Negi    |
| Intern | Akshay Francis |

---

See [WEEKLY_PLAN.md](WEEKLY_PLAN.md) for the phase breakdown.
