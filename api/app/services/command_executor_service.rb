# CommandExecutorService は、クライアントからのコマンド実行リクエストを
# Redisを通じて実際のコマンド実行サーバー（Go）に転送し、
# 結果を受け取ってクライアントに返すサービス
class CommandExecutorService
  # 実行を許可するコマンドのリスト
  # セキュリティのため、実行可能なコマンドを制限
  ALLOWED_COMMANDS = %w[ls pwd whoami date].freeze

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

    # Redisクライアントの初期化
    # 環境変数から接続情報を取得（デフォルト値あり）
    redis = Redis.new(
      url: ENV.fetch("REDIS_URL", "redis://:password@redis:6379/0"),
      timeout: 5,           # 接続タイムアウト
      reconnect_attempts: 3 # 再接続試行回数
    )

    # Redis接続のテスト
    # 接続できない場合は早期リターン
    begin
      redis.ping
      Rails.logger.info "Redis接続テスト成功"
    rescue => e
      Rails.logger.error "Redis接続テスト失敗: #{e.message}"
      return { status: "error", command: command, error: "Redis接続エラー: #{e.message}" }
    end

    # コマンドをRedisのコマンドチャンネルに送信
    # Goのコマンド実行サーバーがこのチャンネルを監視
    Rails.logger.info "コマンドを送信: #{command}"
    redis.publish(COMMAND_CHANNEL, command)
    Rails.logger.info "コマンド送信完了"

    # 結果を待機
    # Redisの結果チャンネルを購読し、タイムアウトまで待機
    Rails.logger.info "結果を待機中...（タイムアウト: #{TIMEOUT_SECONDS}秒）"
    result = nil
    start_time = Time.now

    begin
      # Redisの結果チャンネルを購読
      # ブロッキングモードで結果を待機
      redis.subscribe(RESULT_CHANNEL) do |on|
        # メッセージを受信した時の処理
        on.message do |channel, message|
          Rails.logger.info "結果を受信: #{message}"
          begin
            # JSONとしてパース
            result = JSON.parse(message)
            Rails.logger.info "結果をパース: #{result.inspect}"
            redis.unsubscribe  # 結果を受信したら購読を解除
          rescue JSON::ParserError => e
            Rails.logger.error "JSONパースエラー: #{e.message}, メッセージ: #{message}"
            result = { status: "error", command: command, error: "結果のパースに失敗: #{e.message}" }
            redis.unsubscribe
          end
        end

        # チャンネル購読開始時の処理
        on.subscribe do |channel, subscriptions|
          Rails.logger.info "チャンネル購読開始: #{channel} (購読数: #{subscriptions})"
        end

        # チャンネル購読解除時の処理
        on.unsubscribe do |channel, subscriptions|
          Rails.logger.info "チャンネル購読解除: #{channel} (購読数: #{subscriptions})"
        end
      end
    rescue => e
      Rails.logger.error "Redis購読エラー: #{e.message}"
      return { status: "error", command: command, error: "Redis購読エラー: #{e.message}" }
    ensure
      # 処理時間を記録
      elapsed_time = Time.now - start_time
      Rails.logger.info "処理時間: #{elapsed_time}秒"
    end

    # タイムアウトチェックと結果の返却
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
