package redislock

import "time"

const (
	// redis 命令最大重试次数
	DefaultMaxRetries = 3
	// 默认锁过期时间
	DefaultLockExpireSeconds = 30
	// 默认看门狗工作间隔
	WatchDogWorkStepSeconds = 10
)

type ClientOptions struct {
	maxRetries int // 命令最大重试次数
	// 必填参数
	network  string
	addr     string
	password string
}

type ClientOption func(o *ClientOptions)

func WithMaxRetries(maxRetries int) ClientOption {
	return func(o *ClientOptions) {
		o.maxRetries = maxRetries
	}
}

func repairClient(o *ClientOptions) {
	if o.maxRetries < 0 {
		o.maxRetries = DefaultMaxRetries
	}
}

type LockOptions struct {
	isBlock             bool
	blockWaitingSeconds int64
	expireSeconds       int64
	watchDogMode        bool
}

type LockOption func(c *LockOptions)

func WithBlock() LockOption {
	return func(o *LockOptions) {
		o.isBlock = true
	}
}

func WithBlockWaitingSeconds(waitSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.blockWaitingSeconds = waitSeconds
	}
}

func WithExpireSeconds(expireSeconds int64) LockOption {
	return func(o *LockOptions) {
		o.expireSeconds = expireSeconds
	}
}

func WithWatchDogMode() LockOption {
	return func(o *LockOptions) {
		o.watchDogMode = true
	}
}

func repairLock(o *LockOptions) {
	if o.isBlock && o.blockWaitingSeconds <= 0 {
		// 默认阻塞时间为 5 s
		o.blockWaitingSeconds = 5
	}
	if o.expireSeconds > 0 {
		return
	}
	// 用户未显式指定锁的过期时间，则此时会启动看门狗
	o.expireSeconds = DefaultLockExpireSeconds
	o.watchDogMode = true
}

type SingleNodeConf struct {
	network  string
	addr     string
	password string
	opts     []ClientOption
}

type RedLockOptions struct {
	singleNodesTimeout time.Duration
	expireDuration     time.Duration
}

type RedLockOption func(o *RedLockOptions)

func WithSingleNodesTimeout(singleNodesTimeout time.Duration) RedLockOption {
	return func(o *RedLockOptions) {
		o.singleNodesTimeout = singleNodesTimeout
	}
}

func WithExpireDuration(expireDuration time.Duration) RedLockOption {
	return func(o *RedLockOptions) {
		o.expireDuration = expireDuration
	}
}

func repairRedLock(o *RedLockOptions) {
	if o.singleNodesTimeout <= 0 {
		o.singleNodesTimeout = DefaultSingleLockTimeout
	}
}
