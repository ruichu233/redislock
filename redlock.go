package redislock

import (
	"context"
	"errors"
	"time"
)

const (
	// 红锁中每个节点的默认超时时间为 50 ms
	DefaultSingleLockTimeout = 50
)

type RedLock struct {
	locks []*RedisLock
	RedLockOptions
}

func NewRedLock(key string, confs []*SingleNodeConf, opts ...RedLockOption) (*RedLock, error) {
	if len(confs) < 3 {
		return nil, errors.New("can't use redLock less than 3 nodes")
	}
	r := RedLock{}

	for _, opt := range opts {
		opt(&r.RedLockOptions)
	}
	// 配置兜底
	repairRedLock(&r.RedLockOptions)
	// 要求所有节点累计的超时阈值小于分布式锁的过期时间的十分之一
	if r.expireDuration > 0 && time.Duration(len(confs))*r.singleNodesTimeout*10 > r.expireDuration {
		return nil, errors.New("expire thresholds of single node is too long")
	}

	r.locks = make([]*RedisLock, 0, len(confs))
	for _, conf := range confs {
		client := NewClient(conf.network, conf.addr, conf.password, conf.opts...)
		r.locks = append(r.locks, NewRedisLock(key, client, WithExpireSeconds(int64(r.expireDuration.Seconds()))))
	}
	return &r, nil
}

func (r *RedLock) Lock(ctx context.Context) error {
	succCnt := 0
	for _, lock := range r.locks {
		startTime := time.Now()
		err := lock.Lock(ctx)
		cost := time.Since(startTime)
		if err == nil && cost < r.singleNodesTimeout {
			succCnt++
		}
	}
	if succCnt < len(r.locks)>>1+1 {
		return errors.New("lock failed")
	}
	return nil
}

func (r *RedLock) UnLock(ctx context.Context) error {
	var err error
	for _, lock := range r.locks {
		if _err := lock.UnLock(ctx); _err != nil {
			if err == nil {
				err = _err
			}
		}

	}
	return err
}
