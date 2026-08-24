# stock_alerts.py
import sys
import os
import json
import argparse
import time
import requests
from datetime import datetime

CONFIG_FILE = "alerts.json"
API_KEY = "demo"
BASE_URL = "https://www.alphavantage.co/query"

class Alert:
    def __init__(self, symbol, threshold, above=True, alert_id=None):
        self.id = alert_id or str(uuid.uuid4())[:8]
        self.symbol = symbol.upper()
        self.threshold = threshold
        self.above = above
        self.triggered = False

    def check(self, price):
        if self.above:
            triggered = price > self.threshold
        else:
            triggered = price < self.threshold
        if triggered and not self.triggered:
            self.triggered = True
            return True
        return False

    def to_dict(self):
        return {
            "id": self.id,
            "symbol": self.symbol,
            "threshold": self.threshold,
            "above": self.above,
            "triggered": self.triggered
        }

    @classmethod
    def from_dict(cls, data):
        alert = cls(data["symbol"], data["threshold"], data["above"], data.get("id"))
        alert.triggered = data.get("triggered", False)
        return alert

class StockAlerts:
    def __init__(self):
        self.alerts = []
        self.load()

    def load(self):
        if os.path.exists(CONFIG_FILE):
            with open(CONFIG_FILE, "r") as f:
                data = json.load(f)
                self.alerts = [Alert.from_dict(a) for a in data]

    def save(self):
        with open(CONFIG_FILE, "w") as f:
            json.dump([a.to_dict() for a in self.alerts], f, indent=2)

    def add(self, symbol, threshold, above=True):
        # Remove existing alert for same symbol
        self.alerts = [a for a in self.alerts if a.symbol != symbol.upper()]
        alert = Alert(symbol, threshold, above)
        self.alerts.append(alert)
        self.save()
        print(f"✅ Added alert: {symbol} {'>' if above else '<'} {threshold:.2f}")

    def remove(self, symbol):
        self.alerts = [a for a in self.alerts if a.symbol != symbol.upper()]
        self.save()
        print(f"✅ Removed alert for {symbol.upper()}")

    def list(self):
        if not self.alerts:
            print("No alerts.")
            return
        print("\n📋 Current alerts:")
        for a in self.alerts:
            status = "🔔" if a.triggered else "⏳"
            print(f"  {a.symbol}: {'>' if a.above else '<'} {a.threshold:.2f} {status}")

    def fetch_price(self, symbol):
        params = {
            "function": "GLOBAL_QUOTE",
            "symbol": symbol,
            "apikey": API_KEY
        }
        try:
            resp = requests.get(BASE_URL, params=params, timeout=10)
            data = resp.json()
            quote = data.get("Global Quote", {})
            if not quote:
                return None
            return float(quote.get("05. price", 0))
        except Exception as e:
            print(f"Error fetching {symbol}: {e}", file=sys.stderr)
            return None

    def check_alerts(self):
        for alert in self.alerts:
            price = self.fetch_price(alert.symbol)
            if price is None:
                continue
            if alert.check(price):
                direction = "above" if alert.above else "below"
                print(f"\n🔔 ALERT: {alert.symbol} price is {price:.2f} ({direction} {alert.threshold:.2f})!")
                self.save()

    def watch(self, interval=60):
        print(f"📈 Monitoring alerts every {interval}s (press Ctrl+C to stop)")
        try:
            while True:
                self.check_alerts()
                time.sleep(interval)
        except KeyboardInterrupt:
            print("\n👋 Monitoring stopped.")

def main():
    parser = argparse.ArgumentParser(description="Stock Ticker Alerts")
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    add_parser = subparsers.add_parser("add")
    add_parser.add_argument("symbol")
    add_parser.add_argument("threshold", type=float)
    add_parser.add_argument("--above", action="store_true", default=True)
    add_parser.add_argument("--below", action="store_false", dest="above")

    remove_parser = subparsers.add_parser("remove")
    remove_parser.add_argument("symbol")

    subparsers.add_parser("list")
    watch_parser = subparsers.add_parser("watch")
    watch_parser.add_argument("--interval", type=int, default=60)

    args = parser.parse_args()
    app = StockAlerts()

    if args.cmd == "add":
        app.add(args.symbol, args.threshold, args.above)
    elif args.cmd == "remove":
        app.remove(args.symbol)
    elif args.cmd == "list":
        app.list()
    elif args.cmd == "watch":
        app.watch(args.interval)

if __name__ == "__main__":
    main()
