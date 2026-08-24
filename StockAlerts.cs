// StockAlerts.cs
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net.Http;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading.Tasks;

class Alert
{
    [JsonPropertyName("id")] public string Id { get; set; }
    [JsonPropertyName("symbol")] public string Symbol { get; set; }
    [JsonPropertyName("threshold")] public double Threshold { get; set; }
    [JsonPropertyName("above")] public bool Above { get; set; }
    [JsonPropertyName("triggered")] public bool Triggered { get; set; }

    public Alert() { }
    public Alert(string symbol, double threshold, bool above)
    {
        Id = Guid.NewGuid().ToString().Substring(0,8);
        Symbol = symbol.ToUpper();
        Threshold = threshold;
        Above = above;
        Triggered = false;
    }
}

class App
{
    private List<Alert> alerts = new List<Alert>();
    private readonly string configFile = "alerts.json";
    private readonly JsonSerializerOptions options = new JsonSerializerOptions { WriteIndented = true };
    private static readonly HttpClient client = new HttpClient();
    private const string ApiKey = "demo";
    private const string BaseUrl = "https://www.alphavantage.co/query";

    public App() => Load();

    private void Load()
    {
        if (!File.Exists(configFile)) return;
        string json = File.ReadAllText(configFile);
        alerts = JsonSerializer.Deserialize<List<Alert>>(json) ?? new List<Alert>();
    }

    private void Save()
    {
        string json = JsonSerializer.Serialize(alerts, options);
        File.WriteAllText(configFile, json);
    }

    private Alert GetAlert(string symbol)
        => alerts.FirstOrDefault(a => a.Symbol.Equals(symbol, StringComparison.OrdinalIgnoreCase));

    public void Add(string symbol, double threshold, bool above)
    {
        var existing = GetAlert(symbol);
        if (existing != null) alerts.Remove(existing);
        alerts.Add(new Alert(symbol, threshold, above));
        Save();
        Console.WriteLine($"✅ Added alert: {symbol} {(above ? ">" : "<")} {threshold:F2}");
    }

    public void Remove(string symbol)
    {
        var existing = GetAlert(symbol);
        if (existing != null)
        {
            alerts.Remove(existing);
            Save();
            Console.WriteLine($"✅ Removed alert for {symbol.ToUpper()}");
        }
        else
        {
            Console.WriteLine($"No alert for {symbol.ToUpper()}");
        }
    }

    public void List()
    {
        if (!alerts.Any())
        {
            Console.WriteLine("No alerts.");
            return;
        }
        Console.WriteLine("\n📋 Current alerts:");
        foreach (var a in alerts)
        {
            string status = a.Triggered ? "🔔" : "⏳";
            Console.WriteLine($"  {a.Symbol}: {(a.Above ? ">" : "<")} {a.Threshold:F2} {status}");
        }
    }

    private async Task<double?> FetchPrice(string symbol)
    {
        try
        {
            string url = $"{BaseUrl}?function=GLOBAL_QUOTE&symbol={symbol}&apikey={ApiKey}";
            var response = await client.GetAsync(url);
            response.EnsureSuccessStatusCode();
            string json = await response.Content.ReadAsStringAsync();
            using var doc = JsonDocument.Parse(json);
            var quote = doc.RootElement.GetProperty("Global Quote");
            return quote.GetProperty("05. price").GetDouble();
        }
        catch
        {
            Console.WriteLine($"Error fetching {symbol}");
            return null;
        }
    }

    public async Task CheckAlerts()
    {
        foreach (var a in alerts)
        {
            var price = await FetchPrice(a.Symbol);
            if (!price.HasValue) continue;
            bool triggered = a.Above ? price > a.Threshold : price < a.Threshold;
            if (triggered && !a.Triggered)
            {
                a.Triggered = true;
                string direction = a.Above ? "above" : "below";
                Console.WriteLine($"\n🔔 ALERT: {a.Symbol} price is {price:F2} ({direction} {a.Threshold:F2})!");
                Save();
            }
        }
    }

    public async Task Watch(int interval)
    {
        Console.WriteLine($"📈 Monitoring alerts every {interval}s (press Ctrl+C to stop)");
        while (true)
        {
            await CheckAlerts();
            await Task.Delay(interval * 1000);
        }
    }

    static async Task Main(string[] args)
    {
        if (args.Length < 1)
        {
            Console.WriteLine("Usage: StockAlerts <command> [options]");
            return;
        }
        var app = new App();
        var cmd = args[0];

        switch (cmd)
        {
            case "add":
                if (args.Length < 3) { Console.WriteLine("add <symbol> <threshold> [--above|--below]"); return; }
                string symbol = args[1];
                double threshold = double.Parse(args[2]);
                bool above = true;
                for (int i=3; i<args.Length; i++)
                    if (args[i] == "--below") above = false;
                app.Add(symbol, threshold, above);
                break;
            case "remove":
                if (args.Length < 2) { Console.WriteLine("remove <symbol>"); return; }
                app.Remove(args[1]);
                break;
            case "list":
                app.List();
                break;
            case "watch":
                int interval = 60;
                for (int i=1; i<args.Length; i++)
                    if (args[i] == "--interval" && i+1 < args.Length)
                        interval = int.Parse(args[++i]);
                await app.Watch(interval);
                break;
            default:
                Console.WriteLine("Unknown command.");
                break;
        }
    }
}
