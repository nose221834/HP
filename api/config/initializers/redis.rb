# redisに送信するようのセッション
RedisPublisher = Redis.new(
  url: ENV.fetch("REDIS_URL"),
  timeout: 5,
  reconnect_attempts: 3
)

# redisを購読するようのセッション
# 購読している間、セッションをブロックするため分けて用意している
RedisSubscriber   = Redis.new(
    url: ENV.fetch("REDIS_URL"),
    timeout: 5,
    reconnect_attempts: 3
  )
