package access

import "strings"

// Redis command classification. Unknown commands are treated as admin so a
// read-only connection cannot run them by accident.
var redisReadCommands = map[string]struct{}{
	"GET": {}, "MGET": {}, "GETRANGE": {}, "GETBIT": {}, "STRLEN": {},
	"EXISTS": {}, "TYPE": {}, "TTL": {}, "PTTL": {}, "DUMP": {},
	"KEYS": {}, "SCAN": {}, "RANDOMKEY": {}, "MEMORY": {},
	"HGET": {}, "HMGET": {}, "HGETALL": {}, "HEXISTS": {}, "HKEYS": {},
	"HVALS": {}, "HLEN": {}, "HSCAN": {}, "HSTRLEN": {},
	"LRANGE": {}, "LINDEX": {}, "LLEN": {}, "LPOS": {},
	"SCARD": {}, "SISMEMBER": {}, "SMISMEMBER": {}, "SMEMBERS": {},
	"SSCAN": {}, "SRANDMEMBER": {}, "SDIFF": {}, "SINTER": {}, "SUNION": {},
	"ZCARD": {}, "ZCOUNT": {}, "ZRANGE": {}, "ZRANGEBYSCORE": {}, "ZRANGEBYLEX": {},
	"ZRANK": {}, "ZREVRANK": {}, "ZREVRANGE": {}, "ZREVRANGEBYSCORE": {},
	"ZSCORE": {}, "ZMSCORE": {}, "ZSCAN": {}, "ZLEXCOUNT": {},
	"XLEN": {}, "XRANGE": {}, "XREVRANGE": {}, "XINFO": {}, "XPENDING": {}, "XREAD": {},
	"PING": {}, "ECHO": {}, "INFO": {}, "DBSIZE": {}, "TIME": {}, "LASTSAVE": {},
	"ROLE": {}, "SLOWLOG": {}, "COMMAND": {}, "OBJECT": {}, "TOUCH": {},
	"BITCOUNT": {}, "BITPOS": {}, "GEOHASH": {}, "GEOPOS": {}, "GEODIST": {},
	"GEOSEARCH": {}, "PFCOUNT": {}, "STRALGO": {}, "JSON.GET": {}, "JSON.TYPE": {},
	"JSON.OBJKEYS": {}, "JSON.OBJLEN": {}, "JSON.ARRLEN": {}, "JSON.STRLEN": {},
	"FT.SEARCH": {}, "FT.INFO": {}, "FT.AGGREGATE": {},
}

var redisWriteCommands = map[string]struct{}{
	"SET": {}, "SETNX": {}, "SETEX": {}, "PSETEX": {}, "MSET": {}, "MSETNX": {},
	"APPEND": {}, "INCR": {}, "INCRBY": {}, "INCRBYFLOAT": {}, "DECR": {}, "DECRBY": {},
	"GETSET": {}, "SETBIT": {}, "SETRANGE": {}, "GETDEL": {}, "GETEX": {},
	"DEL": {}, "UNLINK": {}, "EXPIRE": {}, "EXPIREAT": {}, "PEXPIRE": {}, "PEXPIREAT": {},
	"PERSIST": {}, "RENAME": {}, "RENAMENX": {}, "MOVE": {}, "COPY": {}, "RESTORE": {},
	"HSET": {}, "HSETNX": {}, "HMSET": {}, "HINCRBY": {}, "HINCRBYFLOAT": {}, "HDEL": {},
	"LPUSH": {}, "LPUSHX": {}, "RPUSH": {}, "RPUSHX": {}, "LPOP": {}, "RPOP": {},
	"LSET": {}, "LTRIM": {}, "LINSERT": {}, "LMOVE": {}, "RPOPLPUSH": {}, "BLPOP": {},
	"BRPOP": {}, "BRPOPLPUSH": {},
	"SADD": {}, "SREM": {}, "SPOP": {}, "SMOVE": {}, "SDIFFSTORE": {}, "SINTERSTORE": {}, "SUNIONSTORE": {},
	"ZADD": {}, "ZINCRBY": {}, "ZREM": {}, "ZREMRANGEBYRANK": {}, "ZREMRANGEBYSCORE": {},
	"ZREMRANGEBYLEX": {}, "ZUNIONSTORE": {}, "ZINTERSTORE": {}, "ZDIFFSTORE": {}, "ZPOPMIN": {}, "ZPOPMAX": {},
	"XADD": {}, "XDEL": {}, "XTRIM": {}, "XGROUP": {}, "XACK": {}, "XCLAIM": {}, "XAUTOCLAIM": {}, "XREADGROUP": {},
	"PFADD": {}, "PFMERGE": {}, "GEOADD": {}, "BITOP": {},
	"JSON.SET": {}, "JSON.DEL": {}, "JSON.ARRAPPEND": {}, "JSON.ARRINSERT": {},
	"JSON.ARRPOP": {}, "JSON.ARRTRIM": {}, "JSON.NUMINCRBY": {},
}

var redisAdminCommands = map[string]struct{}{
	"FLUSHDB": {}, "FLUSHALL": {}, "SWAPDB": {}, "SELECT": {},
	"CONFIG": {}, "SHUTDOWN": {}, "SLAVEOF": {}, "REPLICAOF": {},
	"DEBUG": {}, "MONITOR": {}, "MIGRATE": {}, "CLUSTER": {},
	"ACL": {}, "MODULE": {}, "SCRIPT": {}, "EVAL": {}, "EVALSHA": {},
	"FUNCTION": {}, "LATENCY": {}, "REPLCONF": {}, "PSYNC": {}, "SYNC": {},
	"BGSAVE": {}, "SAVE": {}, "BGREWRITEAOF": {}, "CLIENT": {},
	"FT.CREATE": {}, "FT.DROPINDEX": {}, "FT.ALTER": {},
}

// ClassifyRedis returns the operation class of a Redis command name.
func ClassifyRedis(command string) Operation {
	cmd := strings.ToUpper(strings.TrimSpace(command))
	if cmd == "" {
		return OpAdmin
	}
	if fields := strings.Fields(cmd); len(fields) > 0 {
		cmd = fields[0]
	}
	if _, ok := redisReadCommands[cmd]; ok {
		return OpRead
	}
	if _, ok := redisWriteCommands[cmd]; ok {
		return OpWrite
	}
	if _, ok := redisAdminCommands[cmd]; ok {
		return OpAdmin
	}
	return OpAdmin
}
