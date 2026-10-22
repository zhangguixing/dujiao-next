package upstream

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/google/uuid"
)

type cardNetManualPingKey struct{}

// Leave headroom below the upstream's rolling per-credential and per-IP limits.
const (
	cardNetRequestSpacing = 1100 * time.Millisecond
	cardNetHostSpacing    = 550 * time.Millisecond
	cardNetWriteSpacing   = 2100 * time.Millisecond
)

type cardNetLimitState struct {
	nextRequest, nextWrite, cooldown time.Time
	reads, writes                    int
	paused                           bool
}

var cardNetLimits = struct {
	sync.Mutex
	states map[string]*cardNetLimitState
}{states: make(map[string]*cardNetLimitState)}

func cardNetKeys(baseURL, apiKey string) (string, string) {
	u, _ := url.Parse(baseURL)
	host := strings.ToLower(baseURL)
	if u != nil {
		host = strings.ToLower(u.Hostname())
	}
	return fmt.Sprintf("upstream:cardnet:credential:%x", sha256.Sum256([]byte(host+"\n"+apiKey))),
		fmt.Sprintf("upstream:cardnet:host:%x", sha256.Sum256([]byte(host)))
}

func cardNetState(key string) *cardNetLimitState {
	state := cardNetLimits.states[key]
	if state == nil {
		state = &cardNetLimitState{}
		cardNetLimits.states[key] = state
	}
	return state
}

func pauseCardNetRequests(baseURL, apiKey string) {
	key, _ := cardNetKeys(baseURL, apiKey)
	cardNetLimits.Lock()
	cardNetState(key).paused = true
	cardNetLimits.Unlock()
}

func (a *CardNetV2Adapter) ResumeRequests() {
	key, _ := cardNetKeys(a.baseURL, a.apiKey)
	cardNetLimits.Lock()
	cardNetState(key).paused = false
	cardNetLimits.Unlock()
}

// Redis coordinates separate API/worker processes. Leases outlive the HTTP
// client's 30s timeout and are removed explicitly after every response.
const cardNetAcquireScript = `
local clock = redis.call('TIME')
local now = clock[1]*1000 + math.floor(clock[2]/1000)
local wait = 0
for _, i in ipairs({1, 2, 6}) do
  wait = math.max(wait, tonumber(redis.call('GET', KEYS[i]) or '0') - now)
end
local lease = KEYS[4]
local capacity = 8
if ARGV[2] == '1' then
  wait = math.max(wait, tonumber(redis.call('GET', KEYS[3]) or '0') - now)
  lease = KEYS[5]
  capacity = 2
end
redis.call('ZREMRANGEBYSCORE', lease, '-inf', now)
if redis.call('ZCARD', lease) >= capacity then wait = math.max(wait, 50) end
if wait > 0 then return wait end
redis.call('SET', KEYS[1], now+550, 'PX', 550)
redis.call('SET', KEYS[2], now+1100, 'PX', 1100)
if ARGV[2] == '1' then redis.call('SET', KEYS[3], now+2100, 'PX', 2100) end
redis.call('ZADD', lease, now+45000, ARGV[1])
redis.call('PEXPIRE', lease, 45000)
return 0`

func acquireCardNetRequest(ctx context.Context, baseURL, apiKey, method string) (func(), error) {
	credential, host := cardNetKeys(baseURL, apiKey)
	write := method != http.MethodGet
	writeFlag := "0"
	if write {
		writeFlag = "1"
	}
	distributed := cache.Enabled()
	manual, _ := ctx.Value(cardNetManualPingKey{}).(bool)
	keys := []string{host + ":next", credential + ":next", credential + ":write", credential + ":reads", credential + ":writes", credential + ":cooldown"}
	token := uuid.NewString()
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cardNetLimits.Lock()
		state, hostState := cardNetState(credential), cardNetState(host)
		if state.paused && !manual {
			cardNetLimits.Unlock()
			return nil, authenticationPausedError{}
		}
		var wait time.Duration
		if !distributed {
			wait = state.reserve(time.Now(), hostState, write)
		} else if state.cooldown.After(time.Now()) {
			wait = time.Until(state.cooldown)
		}
		cardNetLimits.Unlock()
		if distributed && wait == 0 {
			value, err := cache.Eval(ctx, cardNetAcquireScript, keys, token, writeFlag)
			if err != nil {
				return nil, fmt.Errorf("coordinate CardNet requests: %w", err)
			}
			millis, ok := value.(int64)
			if !ok {
				return nil, fmt.Errorf("invalid CardNet limiter response")
			}
			wait = time.Duration(millis) * time.Millisecond
		}
		if wait == 0 {
			return func() {
				if distributed {
					leaseKey := keys[3]
					if write {
						leaseKey = keys[4]
					}
					releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					_, _ = cache.Eval(releaseCtx, `return redis.call('ZREM', KEYS[1], ARGV[1])`, []string{leaseKey}, token)
				} else {
					cardNetLimits.Lock()
					if write {
						state.writes--
					} else {
						state.reads--
					}
					cardNetLimits.Unlock()
				}
			}, nil
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *cardNetLimitState) reserve(now time.Time, host *cardNetLimitState, write bool) time.Duration {
	next := s.nextRequest
	for _, candidate := range []time.Time{host.nextRequest, s.cooldown} {
		if candidate.After(next) {
			next = candidate
		}
	}
	if write && s.nextWrite.After(next) {
		next = s.nextWrite
	}
	if next.After(now) {
		return next.Sub(now)
	}
	if (write && s.writes >= 2) || (!write && s.reads >= 8) {
		return 50 * time.Millisecond
	}
	s.nextRequest = now.Add(cardNetRequestSpacing)
	host.nextRequest = now.Add(cardNetHostSpacing)
	if write {
		s.nextWrite = now.Add(cardNetWriteSpacing)
		s.writes++
	} else {
		s.reads++
	}
	return 0
}

func deferCardNetRequests(ctx context.Context, baseURL, apiKey string, delay time.Duration) {
	if delay <= 0 {
		return
	}
	credential, _ := cardNetKeys(baseURL, apiKey)
	cardNetLimits.Lock()
	state := cardNetState(credential)
	until := time.Now().Add(delay)
	if until.After(state.cooldown) {
		state.cooldown = until
	}
	cardNetLimits.Unlock()
	if cache.Enabled() {
		_, _ = cache.Eval(ctx, `
local ttl = redis.call('PTTL', KEYS[1])
local delay = tonumber(ARGV[1])
if ttl < delay then
 local clock = redis.call('TIME')
 local now = clock[1]*1000 + math.floor(clock[2]/1000)
 redis.call('SET', KEYS[1], now+delay, 'PX', delay)
end
return 1`, []string{credential + ":cooldown"}, delay.Milliseconds())
	}
}

func cardNetRetryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil && date.After(now) {
		return date.Sub(now)
	}
	return 0
}
