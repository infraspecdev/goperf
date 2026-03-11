# Weekly Plan: GoPerf

This document outlines the four phases of the project. Each phase has a goal and expected outcomes. The technical approach is yours to figure out, with guidance from your mentor during weekly planning.

---

## Phase 1: Foundation (Weeks 1-2)

**Goal:** Understand the problem space and build a working prototype that makes concurrent HTTP requests.

### What You Should Do

- Study existing tools. Install `hey` and run it. Read its source code. What does it do? How does it work?
- Learn enough Go to be dangerous. The [Go Tour](https://go.dev/tour/) is a good starting point.
- Build incrementally. First make one HTTP request. Then figure out how to make many at once.
- Get something running on EC2 early, even if it's trivial. Don't wait until the end.

### Questions to Guide You

- What happens when you need to make 100 HTTP requests as fast as possible? What are the options?
- How do you measure how long something takes in Go?
- What's the difference between running things one at a time vs. concurrently?
- How do you know when all your concurrent work is done?
- What can go wrong when multiple things run at the same time?

### By End of Week 2

**Demo:** A CLI that takes a URL, makes concurrent requests for a specified duration, and prints how many requests were made. Running on EC2.

**Be ready to explain:**

- How your code handles concurrency
- What you tried that didn't work
- What you'd do differently

---

## Phase 2: Metrics & Deployment (Weeks 3-4)

**Goal:** Measure latency accurately, report meaningful statistics, and automate your deployment.

### What You Should Do

- Figure out how to measure latency per request and collect the results.
- Learn why averages are misleading for latency. Understand percentiles.
- Set up automated testing and deployment. Code that isn't automatically tested tends to break.
- Push your tool's limits. What happens at high concurrency? Where does it break?

### Questions to Guide You

- When exactly do you start and stop the timer for a request?
- If 99 requests take 10ms and 1 takes 10 seconds, what's the average? What's the p99? Which better represents user experience?
- How do you calculate percentiles? What edge cases exist?
- What's the smallest Docker image you can make? Why might that matter?
- What should happen when your CI pipeline fails?

### By End of Week 4 (MVP Checkpoint)

**Demo:** A working load tester that reports latency percentiles (p50, p90, p99), throughput, and success/failure counts. CI/CD pipeline is green. Running on EC2.

**Be ready to explain:**

- Why percentiles matter more than averages
- Your CI/CD pipeline and what it does
- What happens when you increase concurrency significantly

---

## Phase 3: Features (Weeks 5-6)

**Goal:** Make the tool useful for real-world scenarios.

### What You Should Do

- Support more than just GET requests. Real APIs use POST, PUT, DELETE with headers and bodies.
- Make configuration easier. Typing long commands is tedious. Think about how users will actually use this tool.
- Think about what would make this tool more useful. History? Comparison? Something else?

### Questions to Guide You

- What's the difference between GET and POST from an HTTP perspective? From a load testing perspective?
- How do you handle configuration that's too complex for command-line flags?
- If you run the same test twice, how do you know if performance changed?
- What information would be useful to store? How would you query it later?

### By End of Week 6

**Demo:** Support for different HTTP methods, headers, and request bodies. Configuration from a file. Some form of history or comparison feature. Running on EC2.

**Be ready to explain:**

- Your design decisions for the configuration format
- Trade-offs you made
- What features you chose NOT to build and why

---

## Phase 4: Polish (Weeks 7-8)

**Goal:** Production-quality code and a successful final demo.

### What You Should Do

- Review your own code critically. Is it something you'd be proud to show?
- Fill gaps in testing. What's not tested that should be?
- Write documentation. Can someone else use your tool without asking you questions?
- Prepare your demo. Practice. Know what to do if something fails.

### Questions to Guide You

- If a new developer joined, could they understand your code? What would confuse them?
- What parts of your code are you least confident in? Why?
- What would you do differently if you started over?
- What did you learn that you didn't expect to learn?

### By End of Week 8

**Demo:** A polished, documented tool. You present for 15 minutes: the problem, your solution, a live demo, and what you learned. You handle questions confidently.

**Be ready to explain:**

- Your design and why you made those choices
- What you'd improve with more time
- What you learned about Go, about building software, about yourself

---

## Friday Demos

Every Friday, demo what you built **running on EC2**.

A good demo:

- Shows something working
- Explains what changed since last week
- Identifies what's next
- Calls out blockers or questions

A bad demo:

- "It works on my laptop but I couldn't get it on EC2"
- No explanation of what you learned or struggled with
- Surprises (you should communicate issues before Friday)

---

## Getting Unstuck

When you're stuck:

1. **Define the problem precisely.** "It doesn't work" isn't a problem statement. "I get a 'connection refused' error when running on EC2 but not locally" is.

2. **Research.** Google the exact error message. Read documentation. Look at how other tools solve this.

3. **Experiment.** Try things. Make hypotheses and test them. "I think the issue is X, so if I change Y, Z should happen."

4. **Document what you tried.** When you ask for help, explain: what you're trying to do, what you expected, what happened, and what you already tried.

5. **Ask.** After spending real effort (not 5 minutes, but also not 3 days), ask for help. A well-formed question gets a better answer.

---

## What We Care About

| We care about                        | We don't care about                      |
| ------------------------------------ | ---------------------------------------- |
| You can explain your code            | You used a specific pattern we mentioned |
| The tool works reliably              | The code is "clever"                     |
| You made decisions and defended them | You followed instructions exactly        |
| You improved over 8 weeks            | You got everything right the first time  |
| You asked good questions             | You never needed help                    |

---

## Resources

We're not giving you a comprehensive reading list. Part of the job is figuring out what you need to learn and finding resources yourself.

Starting points:

- [Go Tour](https://go.dev/tour/) — interactive Go tutorial
- [hey source code](https://github.com/rakyll/hey) — a simple load tester written in Go

Beyond that, search, read documentation, and experiment. When you're stuck or need recommendations for learning a specific topic, reach out — we're happy to point you in the right direction.
