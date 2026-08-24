// StockAlerts.java
import java.io.*;
import java.nio.file.*;
import java.net.*;
import java.util.*;
import java.time.*;
import com.google.gson.*;

class Alert {
    String id;
    String symbol;
    double threshold;
    boolean above;
    boolean triggered;

    Alert() {}
    Alert(String symbol, double threshold, boolean above) {
        this.id = UUID.randomUUID().toString().substring(0,8);
        this.symbol = symbol.toUpperCase();
        this.threshold = threshold;
        this.above = above;
        this.triggered = false;
    }
}

public class StockAlerts {
    private List<Alert> alerts = new ArrayList<>();
    private final String configFile = "alerts.json";
    private final Gson gson = new GsonBuilder().setPrettyPrinting().create();
    private static final String API_KEY = "demo";
    private static final String BASE_URL = "https://www.alphavantage.co/query";

    public StockAlerts() { load(); }

    private void load() {
        try {
            Path path = Paths.get(configFile);
            if (Files.exists(path)) {
                String json = new String(Files.readAllBytes(path));
                Alert[] arr = gson.fromJson(json, Alert[].class);
                alerts = Arrays.asList(arr);
            }
        } catch (Exception e) {}
    }

    private void save() {
        try {
            Files.write(Paths.get(configFile), gson.toJson(alerts).getBytes());
        } catch (Exception e) {}
    }

    private Alert getAlert(String symbol) {
        for (Alert a : alerts) {
            if (a.symbol.equalsIgnoreCase(symbol)) return a;
        }
        return null;
    }

    public void add(String symbol, double threshold, boolean above) {
        Alert existing = getAlert(symbol);
        if (existing != null) alerts.remove(existing);
        Alert alert = new Alert(symbol, threshold, above);
        alerts.add(alert);
        save();
        System.out.printf("✅ Added alert: %s %s %.2f%n", symbol, above ? ">" : "<", threshold);
    }

    public void remove(String symbol) {
        Alert existing = getAlert(symbol);
        if (existing != null) {
            alerts.remove(existing);
            save();
            System.out.printf("✅ Removed alert for %s%n", symbol.toUpperCase());
        } else {
            System.out.printf("No alert for %s%n", symbol.toUpperCase());
        }
    }

    public void list() {
        if (alerts.isEmpty()) {
            System.out.println("No alerts.");
            return;
        }
        System.out.println("\n📋 Current alerts:");
        for (Alert a : alerts) {
            String status = a.triggered ? "🔔" : "⏳";
            System.out.printf("  %s: %s %.2f %s%n", a.symbol, a.above ? ">" : "<", a.threshold, status);
        }
    }

    private Double fetchPrice(String symbol) {
        try {
            String url = BASE_URL + "?function=GLOBAL_QUOTE&symbol=" + symbol + "&apikey=" + API_KEY;
            HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("GET");
            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream()));
            StringBuilder sb = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) sb.append(line);
            reader.close();
            JsonObject obj = gson.fromJson(sb.toString(), JsonObject.class);
            JsonObject quote = obj.getAsJsonObject("Global Quote");
            if (quote == null) return null;
            return quote.get("05. price").getAsDouble();
        } catch (Exception e) {
            System.err.println("Error fetching " + symbol + ": " + e.getMessage());
            return null;
        }
    }

    public void checkAlerts() {
        for (Alert a : alerts) {
            Double price = fetchPrice(a.symbol);
            if (price == null) continue;
            boolean triggered = a.above ? price > a.threshold : price < a.threshold;
            if (triggered && !a.triggered) {
                a.triggered = true;
                String direction = a.above ? "above" : "below";
                System.out.printf("\n🔔 ALERT: %s price is %.2f (%s %.2f)!%n", a.symbol, price, direction, a.threshold);
                save();
            }
        }
    }

    public void watch(int interval) throws InterruptedException {
        System.out.printf("📈 Monitoring alerts every %ds (press Ctrl+C to stop)%n", interval);
        while (true) {
            checkAlerts();
            Thread.sleep(interval * 1000L);
        }
    }

    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            System.out.println("Usage: StockAlerts <command> [options]");
            return;
        }
        StockAlerts app = new StockAlerts();
        String cmd = args[0];

        switch (cmd) {
            case "add": {
                if (args.length < 3) { System.out.println("add <symbol> <threshold> [--above|--below]"); return; }
                String symbol = args[1];
                double threshold = Double.parseDouble(args[2]);
                boolean above = true;
                for (int i=3; i<args.length; i++) {
                    if (args[i].equals("--below")) above = false;
                }
                app.add(symbol, threshold, above);
                break;
            }
            case "remove": {
                if (args.length < 2) { System.out.println("remove <symbol>"); return; }
                app.remove(args[1]);
                break;
            }
            case "list":
                app.list();
                break;
            case "watch": {
                int interval = 60;
                for (int i=1; i<args.length; i++) {
                    if (args[i].equals("--interval") && i+1 < args.length) {
                        interval = Integer.parseInt(args[++i]);
                    }
                }
                app.watch(interval);
                break;
            }
            default:
                System.out.println("Unknown command.");
        }
    }
}
