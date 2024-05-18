package redislock

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync/atomic"
	"time"
)

const RedisLockKeyPrefix = "REDIS_LOCK_PREFIX"

var ErrLockAcquiredByOther = errors.New("lock acquired by other")

var ErrNil = redis.Nil

func IsRetryAbleErr(err error) bool {
	return errors.Is(err, ErrLockAcquiredByOther)
}

type RedisLock struct {
	LockOptions
	key    string
	token  string
	client LockClient

	runningDog int32
	stopDog    context.CancelFunc
}

func NewRedisLock(key string, client LockClient, options ...LockOption) *RedisLock {
	l := &RedisLock{
		key:    key,
		token:  GetProcessAndGoroutineIDStr(),
		client: client,
	}
	for _, opt := range options {
		opt(&l.LockOptions)
	}
	repairLock(&l.LockOptions)
	return l
}

func (r *RedisLock) Lock(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			return
		}
		// 加锁成功，开启看门狗
		r.watchDog(ctx)
	}()

	// 尝试获取分布式锁
	err = r.tryLock(ctx)
	if err == nil {
		return nil
	}

	if !r.isBlock {
		return err
	}
	if !IsRetryAbleErr(err) {
		return err
	}
	err = r.blockLock(ctx)
	return err
}

func (r *RedisLock) UnLock(ctx context.Context) error {
	defer func() {
		if r.stopDog != nil {
			r.stopDog()
		}
	}()
	keys := []string{r.getLockKey()}
	values := []interface{}{r.token}
	reply, err := r.client.Even(ctx, LuaCheckAndDeleteDistributedLock, keys, values)
	if err != nil {
		return err
	}
	if ret, _ := reply.(int64); ret != 1 {
		return errors.New("can not unlock without ownership of lock")
	}
	return nil
}

func (r *RedisLock) watchDog(ctx context.Context) {
	// 非开门狗模式
	if !r.watchDogMode {
		return
	}
	// 确保之前看门狗已经关闭
	for atomic.CompareAndSwapInt32(&r.runningDog, 0, 1) {
	}

	ctx, r.stopDog = context.WithCancel(ctx)
	go func() {
		defer func() {
			atomic.StoreInt32(&r.runningDog, 0)
		}()
		r.runWatchDog(ctx)
	}()
}

func (r *RedisLock) runWatchDog(ctx context.Context) {
	ticker := time.NewTicker(WatchDogWorkStepSeconds * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// 定时延长过期时间
		_ = r.DelayExpire(ctx, WatchDogWorkStepSeconds+5)
	}
}

// 基于Lua脚本实现更新过期时间
func (r *RedisLock) DelayExpire(ctx context.Context, expireSeconds int64) error {
	keys := []string{r.getLockKey()}
	args := []interface{}{r.token, expireSeconds}
	reply, err := r.client.Even(ctx, LuaCheckAndExpireDistributedLock, keys, args)
	if err != nil {
		return err
	}

	if ret, _ := reply.(int64); ret != 1 {
		return errors.New("can not expire lock without ownership of lock")
	}
	return nil
}

func (r *RedisLock) getLockKey() string {
	return RedisLockKeyPrefix + r.key
}

func (r *RedisLock) tryLock(ctx context.Context) error {
	// 检查分布式锁是否属于自己
	reply, err := r.client.SetNEX(ctx, r.getLockKey(), r.token, r.expireSeconds)
	if err != nil && !errors.Is(err, ErrNil) {
		return err
	}
	if reply != 1 {
		return fmt.Errorf("reply: %d, err: %w", reply, ErrLockAcquiredByOther)
	}
	return nil
}

func (r *RedisLock) blockLock(ctx context.Context) error {
	// 等待上限
	timeoutChan := time.After(time.Duration(r.blockWaitingSeconds) * time.Second)
	// 轮询 50 ms 一次
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-ctx.Done():
			return fmt.Errorf("lock failed, ctx timeout, err: %w", ctx.Err())
		case <-timeoutChan:
			return fmt.Errorf("block wait timeout, err: %w", ErrLockAcquiredByOther)
		default:
		}
		err := r.tryLock(ctx)
		if err == nil {
			return nil
		}
		if !IsRetryAbleErr(err) {
			return err
		}
	}
	// 不可达
	return nil
}
