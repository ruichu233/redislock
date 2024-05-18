package test

import (
	"context"
	"errors"
	"redislock"
	"sync"
	"testing"
	"time"
)

func Test_Lock(t *testing.T) {
	addr := "47.108.83.210:6379"
	password := "123456"

	client := redislock.NewClient("tcp", addr, password, redislock.WithMaxRetries(3))
	lock1 := redislock.NewRedisLock("test_key", client, redislock.WithExpireSeconds(10))
	lock2 := redislock.NewRedisLock("test_key", client, redislock.WithBlock(), redislock.WithBlockWaitingSeconds(12))

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := lock1.Lock(ctx); err != nil {
			t.Error(err)
			return
		}
	}()
	time.Sleep(2 * time.Second)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := lock2.Lock(ctx); err != nil || errors.Is(err, redislock.ErrLockAcquiredByOther) {
			t.Errorf("got err: %v, expect: %v", err, redislock.ErrLockAcquiredByOther)
			return
		}
	}()

	wg.Wait()
	t.Log("succ")
}
