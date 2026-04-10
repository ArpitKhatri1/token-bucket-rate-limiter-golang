//  TOCKEN  BUCKET RATE LIMITER
// Design requirements
// not distributed , A centeral global token bucket and a global limit
// plus a series of services which have their own request limit
// throttling based on IP?

// For distributed in the future we need to add a central server like a in memory cache like redis, for sharing information between diffrerent rate limiting servers

package main

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Identification of users -> need some sort of key to track the bucket, usually userId or Ip address
// 1. User level rate limit + service level

var store map[string]*GlobalBucket
var globalRate = 3
var storeMutex sync.RWMutex

type GlobalBucket struct { // 1 per user

	capacity    float64
	maxCapacity float64
	// refillType     string // min/secs
	serviceBuckets map[string]*ServiceBucket
	mu             sync.Mutex // giving each bucket its own lock helps user A not blocking user B
}

type ServiceBucket struct { // multiple per user
	maxCapacity float64
	serviceName string
	capacity    float64
	// refillType  string
	rate        uint
	lastRequest time.Time
}

// refill methods -> Active using goroutine, passive (lazy refill)

func (g *GlobalBucket) refill() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.capacity = min(g.maxCapacity, g.capacity+float64(globalRate))
}

func (s *ServiceBucket) refill() {
	now := time.Now()
	duration := now.Sub(s.lastRequest).Seconds()
	tokensToAdd := duration * float64(s.rate)

	// Set the new capacity, but don't go over max
	s.capacity = min(s.maxCapacity, s.capacity+tokensToAdd)
	s.lastRequest = now // IMPORTANT: Update the timestamp so you don't double-refill
}

type RequestBody struct {
	Service string `json:"service"`
}

func handleIncomingRequest(c *gin.Context) {
	// client indentification
	var body RequestBody
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	ip := c.ClientIP()

	storeMutex.Lock()
	val, ok := store[ip]

	if !ok {
		store[ip] = &GlobalBucket{
			capacity:       10,
			maxCapacity:    10,
			serviceBuckets: make(map[string]*ServiceBucket),
		}
		val = store[ip]
	}

	if val.serviceBuckets == nil {
		val.serviceBuckets = make(map[string]*ServiceBucket)
	}

	bucket, ok := val.serviceBuckets[body.Service]

	// 2. Identify the Service bucket
	if !ok {
		bucket = &ServiceBucket{3, body.Service, 3, 1, time.Now()}
		val.serviceBuckets[body.Service] = bucket
	}

	storeMutex.Unlock()
	val.mu.Lock()
	defer val.mu.Unlock() // keep the lock for the entire check and deduct phase
	bucket.refill()

	if bucket.capacity < 1 || val.capacity < 1 {
		// show rate limited error
		c.JSON(429, gin.H{
			"Error": "You are rate limited try again later",
		})
		return
	} else {

		bucket.capacity = bucket.capacity - 1
		val.capacity = val.capacity - 1
		bucket.lastRequest = time.Now()

		// call the resepective endpoint using c.Next() or manual fwd
		c.JSON(200, gin.H{
			"reached": body.Service,
		})
	}

}

func updateGlobalBuckets() {
	for {
		time.Sleep(5 * time.Second)
		storeMutex.RLock()
		for _, bucket := range store {

			bucket.refill()
		}
		storeMutex.RUnlock()

	}

}

func main() {
	store = make(map[string]*GlobalBucket)
	r := gin.Default()

	go updateGlobalBuckets()

	r.GET("/check", handleIncomingRequest)

	r.Run()
}
