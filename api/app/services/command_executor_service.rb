class CommandExecutorService
  ALLOWED_COMMANDS = %w[ls pwd whoami date].freeze
  COMMAND_CHANNEL = "terminal:commands"
  RESULT_CHANNEL = "terminal:results"
  TIMEOUT_SECONDS = 10 # タイムアウト時間を10秒に延長

  def self.execute(command)
    new.execute(command)
  end

  def execute(command)
    Rails.logger.info "コマンド実行開始: #{command}"

    # Redisクライアントの初期化
    redis = Redis.new(
      url: ENV.fetch("REDIS_URL", "redis://:password@redis:6379/0"),
      timeout: 5,
      reconnect_attempts: 3
    )

    # Redis接続のテスト
    begin
      redis.ping
      Rails.logger.info "Redis接続テスト成功"
    rescue => e
      Rails.logger.error "Redis接続テスト失敗: #{e.message}"
      return { status: "error", command: command, error: "Redis接続エラー: #{e.message}" }
    end

    # コマンドを送信
    Rails.logger.info "コマンドを送信: #{command}"
    redis.publish(COMMAND_CHANNEL, command)
    Rails.logger.info "コマンド送信完了"

    # 結果を待機
    Rails.logger.info "結果を待機中...（タイムアウト: #{TIMEOUT_SECONDS}秒）"
    result = nil
    start_time = Time.now

    begin
      redis.subscribe(RESULT_CHANNEL) do |on|
        on.message do |channel, message|
          Rails.logger.info "結果を受信: #{message}"
          begin
            result = JSON.parse(message)
            Rails.logger.info "結果をパース: #{result.inspect}"
            redis.unsubscribe
          rescue JSON::ParserError => e
            Rails.logger.error "JSONパースエラー: #{e.message}, メッセージ: #{message}"
            result = { status: "error", command: command, error: "結果のパースに失敗: #{e.message}" }
            redis.unsubscribe
          end
        end

        on.subscribe do |channel, subscriptions|
          Rails.logger.info "チャンネル購読開始: #{channel} (購読数: #{subscriptions})"
        end

        on.unsubscribe do |channel, subscriptions|
          Rails.logger.info "チャンネル購読解除: #{channel} (購読数: #{subscriptions})"
        end
      end
    rescue => e
      Rails.logger.error "Redis購読エラー: #{e.message}"
      return { status: "error", command: command, error: "Redis購読エラー: #{e.message}" }
    ensure
      elapsed_time = Time.now - start_time
      Rails.logger.info "処理時間: #{elapsed_time}秒"
    end

    if result.nil?
      Rails.logger.error "コマンド実行タイムアウト（#{elapsed_time}秒経過）"
      { status: "error", command: command, error: "コマンド実行タイムアウト（#{elapsed_time}秒経過）" }
    else
      Rails.logger.info "コマンド実行完了: #{result.inspect}"
      result
    end
  end

  private

  def self.allowed_command?(command)
    base_command = command.split(" ").first
    ALLOWED_COMMANDS.include?(base_command)
  end

  def self.sanitize_command(command)
    # コマンドインジェクション対策
    command.gsub(/[;&|`$]/, "")
  end

  def self.error_response(message)
    {
      status: "error",
      error: message
    }
  end
end
