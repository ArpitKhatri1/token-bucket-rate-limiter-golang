//  TOCKEN  BUCKET RATE LIMITER
// Design requirements
// not distributed , A centeral global token bucket and a global limit
// plus a series of services which have their own request limit
// throttling based on IP?

// For distributed in the future we need to add a central server like a in memory cache like redis, for sharing information between diffrerent rate limiting servers

package main

import (
	"fmt"
	"time"
)

// Identification of users -> need some sort of key to track the bucket, usually userId or Ip address
// 1. User level rate limit + service level

type store struct {
	data map[string]GlobalBucket
}

type GlobalBucket struct { // 1 per user
	capacity       float64
	maxCapacity    float64
	refillType     string // min/secs
	rate           uint
	serviceBuckets []ServiceBucket
}

type ServiceBucket struct { // multiple per user
	maxCapacity float64
	serviceName string
	capacity    float64
	refillType  string
	rate        uint
	lastRequest time.Time
}

// refill methods -> Active using goroutine, passive (lazy refill)

func (g *GlobalBucket) refill() {
	g.capacity = min(g.maxCapacity, g.capacity+float64(g.rate))
}

func (s *ServiceBucket) refill() {
	s.capacity += min(s.maxCapacity, float64(s.rate)*time.Since(s.lastRequest).Seconds())
}

func main() {
	fmt.Println("Hi")
}
