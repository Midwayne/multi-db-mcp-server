package redisdb

import (
	"context"
	"fmt"
	"sync"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"

	"github.com/redis/go-redis/v9"
)

type Adapter struct {
	db.Meta
	mu     sync.Mutex
	client *redis.Client
	ready  bool
}

func New(conn spec.Connection) (*Adapter, error) {
	if conn.URI == "" {
		return nil, fmt.Errorf("redis uri is required")
	}
	return &Adapter{Meta: db.Meta{Conn: conn}}, nil
}

func (a *Adapter) EnsureConnected(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ready && a.client != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, a.Conn.ConnectTimeoutDur)
	defer cancel()
	opt, err := redis.ParseURL(a.Conn.URI)
	if err != nil {
		return fmt.Errorf("redis parse uri: %w", err)
	}
	opt.DialTimeout = a.Conn.ConnectTimeoutDur
	opt.ConnMaxIdleTime = a.Conn.InactivityTimeoutDur
	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("redis ping: %w", err)
	}
	a.client = client
	a.ready = true
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if err := a.EnsureConnected(ctx); err != nil {
		return err
	}
	return a.client.Ping(ctx).Err()
}

func (a *Adapter) Close(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var err error
	if a.client != nil {
		err = a.client.Close()
		a.client = nil
	}
	a.ready = false
	return err
}

func (a *Adapter) Connected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ready
}

func (a *Adapter) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key is required")
	}
	val, err := a.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key %q not found", key)
	}
	return val, err
}

func (a *Adapter) Set(ctx context.Context, key, value string) error {
	if !a.Access().Allows(access.OpWrite) {
		return fmt.Errorf("%s", a.Access().DenyMessage(access.OpWrite))
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}
	return a.client.Set(ctx, key, value, 0).Err()
}

func (a *Adapter) Delete(ctx context.Context, keys []string) (int64, error) {
	if !a.Access().Allows(access.OpWrite) {
		return 0, fmt.Errorf("%s", a.Access().DenyMessage(access.OpWrite))
	}
	if len(keys) == 0 {
		return 0, fmt.Errorf("at least one key is required")
	}
	return a.client.Del(ctx, keys...).Result()
}

func (a *Adapter) Scan(ctx context.Context, pattern string, count int64) (map[string]any, error) {
	if pattern == "" {
		pattern = "*"
	}
	max := a.MaxRows()
	var cursor uint64
	var keys []string
	truncated := false
	for {
		batch, next, err := a.client.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if max > 0 && len(keys) >= max {
			keys = keys[:max]
			truncated = true
			break
		}
		if cursor == 0 {
			break
		}
	}
	return map[string]any{
		"keys":      keys,
		"count":     len(keys),
		"truncated": truncated,
		"pattern":   pattern,
	}, nil
}

func (a *Adapter) Info(ctx context.Context, section string) (string, error) {
	if section == "" {
		return a.client.Info(ctx).Result()
	}
	return a.client.Info(ctx, section).Result()
}

func (a *Adapter) Command(ctx context.Context, command string, args []string) (any, error) {
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}
	op := access.ClassifyRedis(command)
	if !a.Access().Allows(op) {
		return nil, fmt.Errorf("%s", a.Access().DenyMessage(op))
	}
	cmdArgs := make([]any, 0, 1+len(args))
	cmdArgs = append(cmdArgs, command)
	for _, arg := range args {
		cmdArgs = append(cmdArgs, arg)
	}
	return a.client.Do(ctx, cmdArgs...).Result()
}

func (a *Adapter) Landscape(ctx context.Context) (any, error) {
	if err := a.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	size, err := a.client.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}
	info, _ := a.client.Info(ctx, "server", "memory").Result()
	return map[string]any{
		"db_size": size,
		"info":    info,
	}, nil
}
