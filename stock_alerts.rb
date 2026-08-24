# stock_alerts.rb
#!/usr/bin/env ruby
require 'json'
require 'net/http'
require 'uri'
require 'time'
require 'securerandom'
require 'optparse'

CONFIG_FILE = 'alerts.json'
API_KEY = 'demo'
BASE_URL = 'https://www.alphavantage.co/query'

class Alert
  attr_accessor :id, :symbol, :threshold, :above, :triggered

  def initialize(symbol, threshold, above = true, id = nil)
    @id = id || SecureRandom.hex(4)
    @symbol = symbol.upcase
    @threshold = threshold
    @above = above
    @triggered = false
  end

  def check(price)
    triggered = @above ? price > @threshold : price < @threshold
    if triggered && !@triggered
      @triggered = true
      return true
    end
    false
  end

  def to_hash
    { id: @id, symbol: @symbol, threshold: @threshold, above: @above, triggered: @triggered }
  end

  def self.from_hash(h)
    alert = new(h['symbol'], h['threshold'], h['above'], h['id'])
    alert.triggered = h['triggered'] || false
    alert
  end
end

class App
  attr_reader :alerts

  def initialize
    @alerts = []
    load
  end

  def load
    if File.exist?(CONFIG_FILE)
      data = JSON.parse(File.read(CONFIG_FILE))
      @alerts = data.map { |h| Alert.from_hash(h) }
    end
  end

  def save
    File.write(CONFIG_FILE, JSON.pretty_generate(@alerts.map(&:to_hash)))
  end

  def add(symbol, threshold, above = true)
    @alerts.reject! { |a| a.symbol == symbol.upcase }
    alert = Alert.new(symbol, threshold, above)
    @alerts << alert
    save
    puts "✅ Added alert: #{symbol} #{above ? '>' : '<'} #{'%.2f' % threshold}"
  end

  def remove(symbol)
    @alerts.reject! { |a| a.symbol == symbol.upcase }
    save
    puts "✅ Removed alert for #{symbol.upcase}"
  end

  def list
    if @alerts.empty?
      puts "No alerts."
      return
    end
    puts "\n📋 Current alerts:"
    @alerts.each do |a|
      status = a.triggered ? '🔔' : '⏳'
      puts "  #{a.symbol}: #{a.above ? '>' : '<'} #{'%.2f' % a.threshold} #{status}"
    end
  end

  def fetch_price(symbol)
    uri = URI("#{BASE_URL}?function=GLOBAL_QUOTE&symbol=#{symbol}&apikey=#{API_KEY}")
    response = Net::HTTP.get(uri)
    data = JSON.parse(response)
    quote = data['Global Quote']
    return nil unless quote
    quote['05. price'].to_f
  rescue => e
    warn "Error fetching #{symbol}: #{e.message}"
    nil
  end

  def check_alerts
    @alerts.each do |alert|
      price = fetch_price(alert.symbol)
      next unless price
      if alert.check(price)
        direction = alert.above ? 'above' : 'below'
        puts "\n🔔 ALERT: #{alert.symbol} price is #{'%.2f' % price} (#{direction} #{'%.2f' % alert.threshold})!"
        save
      end
    end
  end

  def watch(interval = 60)
    puts "📈 Monitoring alerts every #{interval}s (press Ctrl+C to stop)"
    loop do
      check_alerts
      sleep(interval)
    end
  end
end

options = {}
OptionParser.new do |opts|
  opts.banner = "Usage: stock_alerts.rb [command] [options]"
  opts.on("add SYMBOL THRESHOLD", "Add alert") { |v| options[:add] = v }
  opts.on("remove SYMBOL", "Remove alert") { |v| options[:remove] = v }
  opts.on("list", "List alerts") { options[:list] = true }
  opts.on("watch", "Start monitoring") { options[:watch] = true }
  opts.on("--above", "Alert above threshold") { options[:above] = true }
  opts.on("--below", "Alert below threshold") { options[:below] = true }
  opts.on("--interval N", Integer, "Watch interval") { |v| options[:interval] = v }
end.parse!

app = App.new

if options[:add]
  symbol, threshold = options[:add].split
  above = !options[:below]
  app.add(symbol, threshold.to_f, above)
elsif options[:remove]
  app.remove(options[:remove])
elsif options[:list]
  app.list
elsif options[:watch]
  interval = options[:interval] || 60
  app.watch(interval)
else
  puts "Unknown command. Use add, remove, list, watch."
end
