package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedLocker provides distributed locking using Redis
type DistributedLocker struct {
	client   *redis.Client
	workerID string
}

// NewDistributedLocker creates a new distributed locker
func NewDistributedLocker(client *redis.Client, workerID string) *DistributedLocker {
	return &DistributedLocker{
		client:   client,
		workerID: workerID,
	}
}

// AcquireLock attempts to acquire a lock with the given key
func (l *DistributedLocker) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	
	// Try to set the lock with NX (only if not exists)
	result, err := l.client.SetNX(ctx, lockKey, l.workerID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	return result, nil
}

// ReleaseLock releases a lock if held by this worker
func (l *DistributedLocker) ReleaseLock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	
	// Use Lua script to ensure atomic check-and-delete
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	
	_, err := script.Run(ctx, l.client, []string{lockKey}, l.workerID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	
	return nil
}

// RefreshLock extends the TTL of a held lock
func (l *DistributedLocker) RefreshLock(ctx context.Context, key string, ttl time.Duration) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	
	// Use Lua script to ensure atomic check-and-extend
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)
	
	_, err := script.Run(ctx, l.client, []string{lockKey}, l.workerID, ttl.Milliseconds()).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}
	
	return nil
}

// IsLockHeld checks if a lock is currently held by this worker
func (l *DistributedLocker) IsLockHeld(ctx context.Context, key string) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	
	value, err := l.client.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check lock: %w", err)
	}
	
	return value == l.workerID, nil
}

// WaitForLock waits until a lock can be acquired or context is cancelled
func (l *DistributedLocker) WaitForLock(ctx context.Context, key string, ttl time.Duration, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		acquired, err := l.AcquireLock(ctx, key, ttl)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}
		
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Retry
		}
	}
	
	return false, nil
}

// TryLockWithCallback acquires a lock and executes the callback if successful
func (l *DistributedLocker) TryLockWithCallback(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	acquired, err := l.AcquireLock(ctx, key, ttl)
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("failed to acquire lock: %s", key)
	}
	defer l.ReleaseLock(ctx, key)
	
	return fn()
}
