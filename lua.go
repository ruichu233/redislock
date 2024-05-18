package redislock

const LuaCheckAndDeleteDistributedLock = `
local key = KEYS[1]
local target =  ARGV[1]
local getToken = redis.call("GET",key)

if (not getToken or getToken ~= target) then
	return 0
else
	return redis.call("del",key)
end
`

const LuaCheckAndExpireDistributedLock = `
local key = KEYS[1]
local target =  ARGV[1]
local duration = ARGV[2]
local getToken = redis.call("GET",key)

if (not getToken or getToken ~= target) then
	return 0
else
	return redis.call("expire",key,duration)
end
`
