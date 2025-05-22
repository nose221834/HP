# CommandExecutorService は、クライアントからのコマンド実行リクエストを
# Redisを通じて実際のコマンド実行サーバー（Go）に転送し、
# 結果を受け取ってクライアントに返すサービス
class CommandExecutorService
  # 実行を許可するコマンドのリスト
  # セキュリティのため、実行可能なコマンドを制限
  ALLOWED_COMMANDS = %w[ls pwd whoami date cd].freeze

  # Redisのチャンネル名
  # コマンド送信用と結果受信用の2つのチャンネルを使用
  COMMAND_CHANNEL = "terminal:commands"  # コマンドを送信するチャンネル
  RESULT_CHANNEL = "terminal:results"    # 結果を受信するチャンネル
  TIMEOUT_SECONDS = 10  # コマンド実行のタイムアウト時間（秒）

  # クラスメソッドとして実行を提供
  def self.execute(command)
    new.execute(command)
  end

  # コマンド実行のメインロジック
  # 1. Redisに接続
  # 2. コマンドを送信
  # 3. 結果を待機
  # 4. 結果を返却
  def execute(command)
    Rails.logger.info "コマンド実行開始: #{command}"

    # コマンドデータからセッションIDを抽出
    command_data = begin
      JSON.parse(command)
    rescue JSON::ParserError
      { command: command, session_id: nil }
    end

    redis = Redis.new(
      url: ENV.fetch("REDIS_URL", "redis://:password@redis:6379/0"),
      timeout: 5,
      reconnect_attempts: 3
    )

    begin
      redis.ping
      Rails.logger.info "Redis接続テスト成功"
    rescue => e
      Rails.logger.error "Redis接続テスト失敗: #{e.message}"
      return { status: "error", command: command, error: "Redis接続エラー: #{e.message}" }
    end

    # コマンドをJSON形式で送信（クライアントのセッションIDを維持）
    command_json = {
      command: command_data["command"] || command,
      session_id: command_data["session_id"]
    }.to_json

    # 結果を待機するためのキューを作成
    result_queue = Queue.new
    subscription_active = true
    start_time = Time.now

    # 結果を待機するスレッドを開始
    result_thread = Thread.new do
      begin
        # 結果チャンネルを購読
        redis.subscribe(RESULT_CHANNEL) do |on|
          on.message do |channel, message|
            next unless subscription_active

            begin
              parsed_result = JSON.parse(message)
              Rails.logger.info "結果を受信: #{message}"

              # セッションIDが一致する結果のみを処理
              if command_data["session_id"].nil? || parsed_result["session_id"] == command_data["session_id"]
                result_queue.push(parsed_result)
                subscription_active = false
                redis.unsubscribe
              end
            rescue JSON::ParserError => e
              Rails.logger.error "JSONパースエラー: #{e.message}, メッセージ: #{message}"
              if subscription_active
                result_queue.push({ status: "error", command: command, error: "結果のパースに失敗: #{e.message}" })
                subscription_active = false
                redis.unsubscribe
              end
            end
          end

          on.subscribe do |channel, subscriptions|
            Rails.logger.info "チャンネル購読開始: #{channel} (購読数: #{subscriptions})"
            # 購読開始後にコマンドを送信
            redis.publish(COMMAND_CHANNEL, command_json)
            Rails.logger.info "コマンドを送信: #{command_json}"
          end

          on.unsubscribe do |channel, subscriptions|
            Rails.logger.info "チャンネル購読解除: #{channel} (購読数: #{subscriptions})"
          end
        end
      rescue => e
        Rails.logger.error "Redis購読エラー: #{e.message}"
        result_queue.push({ status: "error", command: command, error: "Redis購読エラー: #{e.message}" })
      ensure
        subscription_active = false
      end
    end

    # タイムアウトまで待機
    begin
      result = result_queue.pop(timeout: TIMEOUT_SECONDS)
    rescue ThreadError
      result = nil
    end

    # スレッドの終了を待機
    result_thread.join(0.1)

    elapsed_time = Time.now - start_time
    Rails.logger.info "処理時間: #{elapsed_time}秒"

    if result.nil?
      Rails.logger.error "コマンド実行タイムアウト（#{elapsed_time}秒経過）"
      { status: "error", command: command, error: "コマンド実行タイムアウト（#{elapsed_time}秒経過）" }
    else
      Rails.logger.info "コマンド実行完了: #{result.inspect}"
      result
    end
  end

  private

  # コマンドが許可リストに含まれているかチェック
  def self.allowed_command?(command)
    base_command = command.split(" ").first
    ALLOWED_COMMANDS.include?(base_command)
  end

  # コマンドインジェクション対策
  # 危険な文字を除去
  def self.sanitize_command(command)
    command.gsub(/[;&|`$]/, "")
  end

  # エラーレスポンスの生成
  def self.error_response(message)
    {
      status: "error",
      error: message
    }
  end
end
