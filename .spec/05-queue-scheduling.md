# Phase 5: Queue System & Task Scheduling

**Timeline**: Weeks 14-16  
**Priority**: High - Essential for scalable applications  
**Dependencies**: Phase 1 (Foundation), Phase 2 (Database)

## Overview

Build a robust job queue and task scheduling system inspired by Laravel's Queue and Scheduler with support for:
- Multiple queue drivers (Database, Redis, In-Memory, SQS)
- Job dispatching and processing
- Job chaining and batching
- Failed job handling with retries
- Cron-like task scheduling
- Queue workers and supervisors

## Package Structure

```
queue/
├── manager.go          # Queue manager
├── queue.go           # Queue interface
├── job.go             # Job interface
├── dispatcher.go      # Job dispatcher
├── worker.go          # Queue worker
├── failed_jobs.go     # Failed job tracking
├── chain.go           # Job chaining
├── batch.go           # Job batching
└── events.go          # Queue events

queue/drivers/
├── database.go        # Database queue driver
├── redis.go           # Redis queue driver
├── memory.go          # In-memory driver (testing)
└── sqs.go            # AWS SQS driver

schedule/
├── scheduler.go       # Task scheduler
├── event.go          # Scheduled event
├── cron.go           # Cron expression parser
└── runner.go         # Schedule runner
```

## 1. Job Interface

```go
package queue

// Job represents a queued job
type Job interface {
    Handle() error            // Execute the job
    Failed(error)            // Handle failure
    Queue() string           // Queue name
    Delay() time.Duration    // Delay before processing
    Tries() int              // Max attempts
    Timeout() time.Duration  // Execution timeout
}

// BaseJob provides default implementations
type BaseJob struct {
    queue   string
    delay   time.Duration
    tries   int
    timeout time.Duration
}

func (j *BaseJob) Queue() string           { return j.queue }
func (j *BaseJob) Delay() time.Duration    { return j.delay }
func (j *BaseJob) Tries() int              { return j.tries }
func (j *BaseJob) Timeout() time.Duration  { return j.timeout }
func (j *BaseJob) Failed(err error)        { /* default: log error */ }
```

## 2. Example Jobs

### Send Email Job

```go
package jobs

type SendEmailJob struct {
    queue.BaseJob
    To      string
    Subject string
    Body    string
}

func (j *SendEmailJob) Handle() error {
    mailer := container.Resolve[mail.Mailer]()
    return mailer.Send(j.To, j.Subject, j.Body)
}

func (j *SendEmailJob) Failed(err error) {
    log.Error("Failed to send email", "to", j.To, "error", err)
}
```

### Process Video Job

```go
package jobs

type ProcessVideoJob struct {
    queue.BaseJob
    VideoID uint
}

func (j *ProcessVideoJob) Handle() error {
    video := &models.Video{}
    db.First(video, j.VideoID)
    
    // Process video
    if err := processVideo(video); err != nil {
        return err
    }
    
    video.Status = "processed"
    return db.Save(video).Error
}

func (j *ProcessVideoJob) Tries() int {
    return 3  // Retry up to 3 times
}

func (j *ProcessVideoJob) Timeout() time.Duration {
    return 10 * time.Minute
}
```

## 3. Queue Drivers

### Queue Interface

```go
type Queue interface {
    Push(job Job) error
    PushOn(queue string, job Job) error
    Later(delay time.Duration, job Job) error
    Pop(queue string) (*JobPayload, error)
    Delete(id string) error
    Release(id string, delay time.Duration) error
    Size(queue string) int
}

type JobPayload struct {
    ID       string
    Job      Job
    Attempts int
    Queue    string
}
```

### Database Driver

Stores jobs in database table:

```go
type DatabaseQueue struct {
    db *gorm.DB
}

type QueueJob struct {
    ID           string    `gorm:"primaryKey;type:varchar(36)"`
    Queue        string    `gorm:"index;type:varchar(191)"`
    Payload      []byte    `gorm:"type:text"`
    Attempts     int       `gorm:"default:0"`
    ReservedAt   *time.Time
    AvailableAt  time.Time `gorm:"index"`
    CreatedAt    time.Time
}

func (q *DatabaseQueue) Push(job Job) error {
    return q.PushOn("default", job)
}

func (q *DatabaseQueue) PushOn(queue string, job Job) error {
    payload, err := serialize(job)
    if err != nil {
        return err
    }
    
    queueJob := &QueueJob{
        ID:          uuid.New().String(),
        Queue:       queue,
        Payload:     payload,
        AvailableAt: time.Now().Add(job.Delay()),
    }
    
    return q.db.Create(queueJob).Error
}

func (q *DatabaseQueue) Pop(queue string) (*JobPayload, error) {
    var queueJob QueueJob
    
    // Lock and get next available job
    err := q.db.Transaction(func(tx *gorm.DB) error {
        err := tx.Where("queue = ?", queue).
            Where("available_at <= ?", time.Now()).
            Where("reserved_at IS NULL").
            Order("available_at ASC").
            First(&queueJob).Error
        
        if err != nil {
            return err
        }
        
        // Reserve the job
        now := time.Now()
        queueJob.ReservedAt = &now
        queueJob.Attempts++
        
        return tx.Save(&queueJob).Error
    })
    
    if err != nil {
        return nil, err
    }
    
    job, err := deserialize(queueJob.Payload)
    if err != nil {
        return nil, err
    }
    
    return &JobPayload{
        ID:       queueJob.ID,
        Job:      job,
        Attempts: queueJob.Attempts,
        Queue:    queueJob.Queue,
    }, nil
}

func (q *DatabaseQueue) Delete(id string) error {
    return q.db.Where("id = ?", id).Delete(&QueueJob{}).Error
}

func (q *DatabaseQueue) Release(id string, delay time.Duration) error {
    return q.db.Model(&QueueJob{}).
        Where("id = ?", id).
        Updates(map[string]interface{}{
            "reserved_at":  nil,
            "available_at": time.Now().Add(delay),
        }).Error
}
```

### Redis Driver

Uses Redis lists for high performance:

```go
type RedisQueue struct {
    client *redis.Client
}

func (q *RedisQueue) Push(job Job) error {
    return q.PushOn("default", job)
}

func (q *RedisQueue) PushOn(queue string, job Job) error {
    payload, err := serialize(job)
    if err != nil {
        return err
    }
    
    key := fmt.Sprintf("queues:%s", queue)
    
    if job.Delay() > 0 {
        // Use sorted set for delayed jobs
        score := time.Now().Add(job.Delay()).Unix()
        return q.client.ZAdd(ctx, key+":delayed", redis.Z{
            Score:  float64(score),
            Member: payload,
        }).Err()
    }
    
    return q.client.RPush(ctx, key, payload).Err()
}

func (q *RedisQueue) Pop(queue string) (*JobPayload, error) {
    key := fmt.Sprintf("queues:%s", queue)
    
    // Move delayed jobs to main queue
    q.migrateDelayedJobs(queue)
    
    // Pop job from queue
    result, err := q.client.LPop(ctx, key).Result()
    if err == redis.Nil {
        return nil, nil // No jobs
    }
    if err != nil {
        return nil, err
    }
    
    job, err := deserialize([]byte(result))
    if err != nil {
        return nil, err
    }
    
    return &JobPayload{
        ID:    uuid.New().String(),
        Job:   job,
        Queue: queue,
    }, nil
}

func (q *RedisQueue) migrateDelayedJobs(queue string) {
    key := fmt.Sprintf("queues:%s:delayed", queue)
    now := float64(time.Now().Unix())
    
    // Get expired delayed jobs
    jobs, _ := q.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
        Min: "0",
        Max: fmt.Sprintf("%f", now),
    }).Result()
    
    // Move to main queue
    for _, job := range jobs {
        q.client.RPush(ctx, fmt.Sprintf("queues:%s", queue), job)
        q.client.ZRem(ctx, key, job)
    }
}
```

## 4. Queue Worker

Processes jobs from queues:

```go
type Worker struct {
    manager    *Manager
    queues     []string
    maxTries   int
    timeout    time.Duration
    sleep      time.Duration
    stopChan   chan struct{}
    failedJobs *FailedJobRepository
}

func NewWorker(manager *Manager, queues []string) *Worker {
    return &Worker{
        manager:    manager,
        queues:     queues,
        maxTries:   3,
        timeout:    60 * time.Second,
        sleep:      3 * time.Second,
        stopChan:   make(chan struct{}),
        failedJobs: NewFailedJobRepository(),
    }
}

func (w *Worker) Start() {
    for {
        select {
        case <-w.stopChan:
            return
        default:
            w.processNextJob()
        }
    }
}

func (w *Worker) Stop() {
    close(w.stopChan)
}

func (w *Worker) processNextJob() {
    // Try each queue
    for _, queue := range w.queues {
        payload, err := w.manager.Queue().Pop(queue)
        if err != nil {
            log.Error("Failed to pop job", "error", err)
            continue
        }
        
        if payload == nil {
            continue // No jobs
        }
        
        // Process job
        if err := w.runJob(payload); err != nil {
            w.handleFailedJob(payload, err)
        } else {
            // Delete successful job
            w.manager.Queue().Delete(payload.ID)
        }
        
        return // Processed one job
    }
    
    // No jobs found, sleep
    time.Sleep(w.sleep)
}

func (w *Worker) runJob(payload *JobPayload) error {
    // Create timeout context
    ctx, cancel := context.WithTimeout(context.Background(), 
        payload.Job.Timeout())
    defer cancel()
    
    // Run job in goroutine with timeout
    errChan := make(chan error, 1)
    go func() {
        errChan <- payload.Job.Handle()
    }()
    
    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        return errors.New("job timeout")
    }
}

func (w *Worker) handleFailedJob(payload *JobPayload, err error) {
    if payload.Attempts < payload.Job.Tries() {
        // Retry with exponential backoff
        delay := time.Duration(math.Pow(2, float64(payload.Attempts))) * time.Second
        w.manager.Queue().Release(payload.ID, delay)
        log.Warn("Job failed, retrying",
            "attempts", payload.Attempts,
            "delay", delay,
            "error", err,
        )
    } else {
        // Max retries exceeded
        payload.Job.Failed(err)
        w.failedJobs.Log(payload, err)
        w.manager.Queue().Delete(payload.ID)
        log.Error("Job failed permanently", "error", err)
    }
}
```

## 5. Job Dispatching

### Dispatcher

```go
type Dispatcher struct {
    manager *Manager
}

func Dispatch(job Job) *PendingDispatch {
    return &PendingDispatch{
        job:     job,
        manager: container.Resolve[*Manager](),
    }
}

type PendingDispatch struct {
    job     Job
    manager *Manager
    queue   string
    delay   time.Duration
}

func (pd *PendingDispatch) OnQueue(queue string) *PendingDispatch {
    pd.queue = queue
    return pd
}

func (pd *PendingDispatch) Delay(delay time.Duration) *PendingDispatch {
    pd.delay = delay
    return pd
}

func (pd *PendingDispatch) Dispatch() error {
    if pd.delay > 0 {
        return pd.manager.Queue().Later(pd.delay, pd.job)
    }
    if pd.queue != "" {
        return pd.manager.Queue().PushOn(pd.queue, pd.job)
    }
    return pd.manager.Queue().Push(pd.job)
}

// Usage
Dispatch(&SendEmailJob{
    To:      "user@example.com",
    Subject: "Welcome",
    Body:    "Welcome to our app!",
}).OnQueue("emails").Delay(5 * time.Minute).Dispatch()
```

## 6. Job Chaining

Execute jobs sequentially:

```go
type Chain struct {
    jobs []Job
}

func NewChain(jobs ...Job) *Chain {
    return &Chain{jobs: jobs}
}

func (c *Chain) Dispatch() error {
    if len(c.jobs) == 0 {
        return nil
    }
    
    // Wrap jobs to execute next on success
    for i := 0; i < len(c.jobs)-1; i++ {
        current := c.jobs[i]
        next := c.jobs[i+1]
        
        // Create chained job
        c.jobs[i] = &ChainedJob{
            Job:  current,
            Next: next,
        }
    }
    
    // Dispatch first job
    return Dispatch(c.jobs[0]).Dispatch()
}

type ChainedJob struct {
    Job  Job
    Next Job
}

func (cj *ChainedJob) Handle() error {
    // Execute current job
    if err := cj.Job.Handle(); err != nil {
        return err
    }
    
    // Dispatch next job
    return Dispatch(cj.Next).Dispatch()
}

// Usage
Chain(
    &ProcessVideo{VideoID: 1},
    &OptimizeVideo{VideoID: 1},
    &NotifyUser{VideoID: 1},
).Dispatch()
```

## 7. Job Batching

Track groups of related jobs:

```go
type Batch struct {
    ID       string
    Name     string
    Jobs     []Job
    Total    int
    Pending  int
    Failed   int
    Finished int
    onSuccess func()
    onComplete func()
}

func NewBatch(name string, jobs []Job) *Batch {
    return &Batch{
        ID:      uuid.New().String(),
        Name:    name,
        Jobs:    jobs,
        Total:   len(jobs),
        Pending: len(jobs),
    }
}

func (b *Batch) Then(callback func()) *Batch {
    b.onSuccess = callback
    return b
}

func (b *Batch) Finally(callback func()) *Batch {
    b.onComplete = callback
    return b
}

func (b *Batch) Dispatch() error {
    // Save batch metadata
    batchRepo.Save(b)
    
    // Dispatch all jobs
    for _, job := range b.Jobs {
        wrapped := &BatchedJob{
            Job:     job,
            BatchID: b.ID,
        }
        Dispatch(wrapped).Dispatch()
    }
    
    return nil
}

// Usage
Batch("Process Users", []Job{
    &ProcessUser{ID: 1},
    &ProcessUser{ID: 2},
    &ProcessUser{ID: 3},
}).Then(func() {
    // All succeeded
    log.Info("All users processed")
}).Finally(func() {
    // Always runs
    log.Info("Batch completed")
}).Dispatch()
```

## 8. Task Scheduling

Cron-like task scheduling:

```go
package schedule

type Scheduler struct {
    events []*Event
}

type Event struct {
    command    func() error
    expression string
    timezone   *time.Location
    before     []func()
    after      []func()
}

func NewScheduler() *Scheduler {
    return &Scheduler{events: []*Event{}}
}

// Schedule a callback
func (s *Scheduler) Call(callback func() error) *Event {
    event := &Event{command: callback}
    s.events = append(s.events, event)
    return event
}

// Schedule a job
func (s *Scheduler) Job(job queue.Job) *Event {
    return s.Call(func() error {
        return queue.Dispatch(job).Dispatch()
    })
}

// Fluent API
func (e *Event) Daily() *Event {
    return e.Cron("0 0 * * *")
}

func (e *Event) DailyAt(time string) *Event {
    parts := strings.Split(time, ":")
    return e.Cron(fmt.Sprintf("%s %s * * *", parts[1], parts[0]))
}

func (e *Event) Hourly() *Event {
    return e.Cron("0 * * * *")
}

func (e *Event) EveryMinute() *Event {
    return e.Cron("* * * * *")
}

func (e *Event) EveryFiveMinutes() *Event {
    return e.Cron("*/5 * * * *")
}

func (e *Event) Weekly() *Event {
    return e.Cron("0 0 * * 0")
}

func (e *Event) Monthly() *Event {
    return e.Cron("0 0 1 * *")
}

func (e *Event) Cron(expression string) *Event {
    e.expression = expression
    return e
}

func (e *Event) Timezone(tz string) *Event {
    loc, _ := time.LoadLocation(tz)
    e.timezone = loc
    return e
}

func (e *Event) Before(callback func()) *Event {
    e.before = append(e.before, callback)
    return e
}

func (e *Event) After(callback func()) *Event {
    e.after = append(e.after, callback)
    return e
}

// Check if event is due
func (e *Event) IsDue(now time.Time) bool {
    if e.timezone != nil {
        now = now.In(e.timezone)
    }
    
    schedule, _ := cron.Parse(e.expression)
    next := schedule.Next(now.Add(-1 * time.Minute))
    
    return now.After(next) && now.Before(next.Add(1*time.Minute))
}

// Run scheduler
func (s *Scheduler) Run() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case now := <-ticker.C:
            s.runDueEvents(now)
        }
    }
}

func (s *Scheduler) runDueEvents(now time.Time) {
    for _, event := range s.events {
        if event.IsDue(now) {
            go s.runEvent(event)
        }
    }
}

func (s *Scheduler) runEvent(event *Event) {
    // Before callbacks
    for _, before := range event.before {
        before()
    }
    
    // Run command
    if err := event.command(); err != nil {
        log.Error("Scheduled task failed", "error", err)
    }
    
    // After callbacks
    for _, after := range event.after {
        after()
    }
}
```

### Usage

```go
// In application bootstrap
scheduler := schedule.NewScheduler()

// Schedule jobs
scheduler.Job(&SendNewsletterJob{}).Weekly().Mondays().At("09:00")

scheduler.Job(&CleanupLogsJob{}).Daily()

scheduler.Job(&BackupDatabaseJob{}).
    DailyAt("01:00").
    Timezone("America/New_York")

scheduler.Call(func() error {
    // Custom logic
    return cleanupTempFiles()
}).EveryFiveMinutes()

// Run scheduler
go scheduler.Run()
```

## 9. CLI Commands

```bash
# Start queue worker
glib queue:work

# Specific queue
glib queue:work --queue=emails

# Multiple queues with priority
glib queue:work --queue=high,default,low

# Max tries
glib queue:work --tries=3

# Timeout
glib queue:work --timeout=60

# Number of workers
glib queue:work --workers=4

# List failed jobs
glib queue:failed

# Retry failed job
glib queue:retry <job-id>

# Retry all failed jobs
glib queue:retry --all

# Delete failed job
glib queue:forget <job-id>

# Flush failed jobs
glib queue:flush

# Start scheduler
glib schedule:work

# Run scheduled tasks once
glib schedule:run
```

## Success Metrics

### Phase 5 Complete When:

- ✅ Jobs dispatch to multiple queue drivers
- ✅ Workers process jobs reliably
- ✅ Failed jobs tracked and can be retried
- ✅ Job chaining works
- ✅ Batching tracks job groups
- ✅ Scheduler runs cron-like tasks
- ✅ All drivers (database, Redis, memory) work
- ✅ CLI commands manage queues
- ✅ Tests pass with >90% coverage
- ✅ Documentation complete
